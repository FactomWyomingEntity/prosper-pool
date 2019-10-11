package stratum

import (
	"crypto/rand"
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

// AddMiner will add a miner to the map, and return a unique session id
func (m *MinerMap) AddMiner(u *Miner) string {
	session := make([]byte, 16)
	_, _ = rand.Read(session)
	u.sessionID = fmt.Sprintf("%x", session)
	u.joined = time.Now()
	m.Lock()
	u.extraNonce1 = m.nextNonce
	m.nextNonce++
	m.miners[u.sessionID] = u
	m.Unlock()
	return u.sessionID
}
