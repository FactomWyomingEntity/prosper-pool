// Copyright (c) of parts are held by the various contributors (see the CLA)
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package mining

import (
	"context"
	"fmt"

	"github.com/pegnet/pegnet/opr"
	log "github.com/sirupsen/logrus"
)

const (
	_ = iota
	BatchCommand
	NewOPRHash
	ResetRecords
	MinimumAccept
	PauseMining
	ResumeMining
)

type MinerCommand struct {
	Command int
	Data    interface{}
}

type Winner struct {
	OPRHash string
	Nonce   string
}

// PegnetMiner mines an OPRhash
type PegnetMiner struct {
	// ID is the miner number, starting with "1".
	ID int `json:"id"`

	// Miner commands
	commands chan *MinerCommand

	successes chan *Winner

	// All the state variables PER oprhash.
	MiningState oprMiningState

	// Tells us we are paused
	paused bool
}

type oprMiningState struct {
	// Used to compute new hashes
	oprhash []byte

	// Used to track noncing
	*NonceIncrementer

	// Used to return hashes
	minimumDifficulty uint64
}

// NonceIncrementer is just simple to increment nonces
type NonceIncrementer struct {
	Nonce         []byte
	lastNonceByte int
}

func NewNonceIncrementer(id int) *NonceIncrementer {
	n := new(NonceIncrementer)
	n.Nonce = []byte{byte(id), 0}
	n.lastNonceByte = 1
	return n
}

// NextNonce is just counting to get the next nonce. We preserve
// the first byte, as that is our ID and give us our nonce space
//	So []byte(ID, 255) -> []byte(ID, 1, 0) -> []byte(ID, 1, 1)
func (i *NonceIncrementer) NextNonce() {
	idx := len(i.Nonce) - 1
	for {
		i.Nonce[idx]++
		if i.Nonce[idx] == 0 {
			idx--
			if idx == 0 { // This is my prefix, don't touch it!
				rest := append([]byte{1}, i.Nonce[1:]...)
				i.Nonce = append([]byte{i.Nonce[0]}, rest...)
				break
			}
		} else {
			break
		}
	}

}

func (p *PegnetMiner) ResetNonce() {
	p.MiningState.NonceIncrementer = NewNonceIncrementer(p.ID)
}

func NewPegnetMiner(id int, commands chan *MinerCommand, successes chan *Winner) *PegnetMiner {
	p := new(PegnetMiner)
	p.ID = id
	p.commands = commands
	p.successes = successes

	// Some inits
	p.MiningState.NonceIncrementer = NewNonceIncrementer(p.ID)
	p.ResetNonce()

	return p
}

func (p *PegnetMiner) IsPaused() bool {
	return p.paused
}

func (p *PegnetMiner) Mine(ctx context.Context) {
	mineLog := log.WithFields(log.Fields{"miner": p.ID})
	var _ = mineLog
	select {
	// Wait for the first command to start
	// We start 'paused'. Any command will knock us out of this init phase
	case c := <-p.commands:
		p.HandleCommand(c)
	case <-ctx.Done():
		return // Cancelled
	}

	for {
		select {
		case <-ctx.Done():
			return // Mining cancelled
		case c := <-p.commands:
			p.HandleCommand(c)
		default:
		}

		if len(p.MiningState.oprhash) == 0 {
			p.paused = true
		}

		if p.paused {
			// Waiting on a resume command
			p.waitForResume(ctx)
			continue
		}

		p.MiningState.NextNonce()

		diff := opr.ComputeDifficulty(p.MiningState.oprhash, p.MiningState.Nonce)
		if diff > p.MiningState.minimumDifficulty {
			success := &Winner{
				OPRHash: fmt.Sprintf("%x", p.MiningState.oprhash),
				Nonce:   fmt.Sprintf("%x", p.MiningState.Nonce),
			}
			p.successes <- success
		}
	}

}

func (p *PegnetMiner) SendCommand(mc *MinerCommand) {
	p.commands <- mc
}

func (p *PegnetMiner) HandleCommand(c *MinerCommand) {
	switch c.Command {
	case BatchCommand:
		commands := c.Data.([]*MinerCommand)
		for _, c := range commands {
			p.HandleCommand(c)
		}
	case NewOPRHash:
		p.MiningState.oprhash = c.Data.([]byte)
	case ResetRecords:
		p.ResetNonce()
	case MinimumAccept:
		p.MiningState.minimumDifficulty = c.Data.(uint64)
	case PauseMining:
		p.paused = true
	case ResumeMining:
		p.paused = false
	}
}

func (p *PegnetMiner) waitForResume(ctx context.Context) {
	// Pause until we get a new start or are cancelled
	for {
		select {
		case <-ctx.Done(): // Mining cancelled
			return
		case c := <-p.commands:
			p.HandleCommand(c)
			if !p.paused {
				return
			}
		}
	}
}

// CommandBuilder just let's me use building syntax to build commands
type CommandBuilder struct {
	command  *MinerCommand
	commands []*MinerCommand
}

func BuildCommand() *CommandBuilder {
	c := new(CommandBuilder)
	c.command = new(MinerCommand)
	c.command.Command = BatchCommand
	return c
}

func (b *CommandBuilder) NewOPRHash(oprhash []byte) *CommandBuilder {
	b.commands = append(b.commands, &MinerCommand{Command: NewOPRHash, Data: oprhash})
	return b
}

func (b *CommandBuilder) ResetRecords() *CommandBuilder {
	b.commands = append(b.commands, &MinerCommand{Command: ResetRecords, Data: nil})
	return b
}

func (b *CommandBuilder) MinimumDifficulty(min uint64) *CommandBuilder {
	b.commands = append(b.commands, &MinerCommand{Command: MinimumAccept, Data: min})
	return b
}

func (b *CommandBuilder) PauseMining() *CommandBuilder {
	b.commands = append(b.commands, &MinerCommand{Command: PauseMining, Data: nil})
	return b
}

func (b *CommandBuilder) ResumeMining() *CommandBuilder {
	b.commands = append(b.commands, &MinerCommand{Command: ResumeMining, Data: nil})
	return b
}

func (b *CommandBuilder) Build() *MinerCommand {
	b.command.Data = b.commands
	return b.command
}
