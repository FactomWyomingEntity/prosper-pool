package stratum

import (
	crand "crypto/rand"
	"encoding/binary"
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
	buf := make([]byte, 4)
	_, _ = crand.Read(buf)
	m.nextNonce = binary.BigEndian.Uint32(buf)
	if m.nextNonce == 0 {
		m.nextNonce = uint32(time.Now().UnixNano())
	}

	return m
}

type NotifyError struct {
	Session string
	Error   error
}

func (m *MinerMap) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.miners)
}

func (m *MinerMap) Notify(msg json.RawMessage) []NotifyError {
	var errs []NotifyError
	m.RLock()
	for session, m := range m.miners {
		m.ResetNonceHistory()
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
	_, _ = crand.Read(session)
	u.sessionID = fmt.Sprintf("%x", session)
	u.joined = time.Now()
	m.Lock()
	u.nonce = m.nextNonce
	m.nextNonce++
	m.miners[u.sessionID] = u
	m.Unlock()
	return u.sessionID
}

func (m *MinerMap) DisconnectMiner(u *Miner) {
	m.Lock()
	defer m.Unlock()

	delete(m.miners, u.sessionID)
	// Close the connection if they are still listening.
	u.conn.Close()
}

// GetMiner returns a pointer to the miner in the MinerMap under the 'name' key
func (m *MinerMap) GetMiner(name string) (*Miner, error) {
	m.Lock()
	defer m.Unlock()

	if miner, ok := m.miners[name]; ok {
		return miner, nil
	} else {
		return nil, fmt.Errorf("No client/miner named %s", name)
	}
}

func (m *MinerMap) ListMiners() []string {
	names := make([]string, 0)
	m.Lock()
	for miner := range m.miners {
		names = append(names, miner)
	}
	m.Unlock()
	return names
}

func (m *MinerMap) SnapShot() []MinerSnapShot {
	m.Lock()
	defer m.Unlock()

	snaps := make([]MinerSnapShot, len(m.miners))
	var c int
	for _, v := range m.miners {
		snaps[c] = v.SnapShot()
		c++
	}

	return snaps
}
