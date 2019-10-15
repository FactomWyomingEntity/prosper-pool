package stratum

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// TODO: Benchmark this for many miners.
type MinerMap struct {
	miners map[string]*Miner
	// nextNonce is the nonce to hand out to the next miner
	// This sets their search space
	nextNonce uint32
	sync.RWMutex
}

func NewMinerMap() *MinerMap {
	m := new(MinerMap)
	m.miners = make(map[string]*Miner)
	// Seed the nonce
	m.nextNonce = uint32(time.Now().UnixNano())
	return m
}

type NotifyError struct {
	Session string
	Error   error
}

func (m MinerMap) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.miners)
}

func (m MinerMap) Notify(msg json.RawMessage) []NotifyError {
	var errs []NotifyError
	m.RLock()
	for session, m := range m.miners {
		err := m.Broadcast(msg)
		if err != nil {
			errs = append(errs, NotifyError{
				Session: session,
				Error:   err,
			})
		}
	}
	m.RUnlock()
	return errs
}

// AddMiner will add a miner to the map, and return a unique session id
func (m *MinerMap) AddMiner(u *Miner) string {
	session := make([]byte, 16)
	_, _ = rand.Read(session)
	u.sessionID = fmt.Sprintf("%x", session)
	u.sessionID = "abc"
	u.joined = time.Now()
	m.Lock()
	u.nonce = m.nextNonce
	m.nextNonce++
	m.miners[u.sessionID] = u
	m.Unlock()
	return u.sessionID
}

// GetMiner will add a miner to the map, and return a unique session id
func (m *MinerMap) GetMiner(name string) (*Miner, error) {
	if miner, ok := m.miners[name]; ok {
		return miner, nil
	} else {
		return nil, fmt.Errorf("No client/miner named %s", name)
	}
}
