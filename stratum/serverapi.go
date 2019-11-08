package stratum

// MinerSnapShot is for the admins to get a glimpse at the set of miners on
// the stratum server. Since the miners are active connections, we will save
// a snapshot.
// TODO: This should be replaced with a better api, but atm we need some insight
func (s *Server) MinersSnapShot() []MinerSnapShot {
	return s.Miners.SnapShot()
}

type MinerSnapShot struct {
	IP              string
	SessionID       string
	PrefferedTarget uint64
	Subscribed      bool
	Nonce           uint32
	Agent           string // Agent/version from subscribe
	Username        string
	Minerid         string
	Authorized      bool
}

func (m *Miner) SnapShot() (snap MinerSnapShot) {
	return MinerSnapShot{
		IP:              m.conn.RemoteAddr().String(),
		SessionID:       m.sessionID,
		PrefferedTarget: m.preferredTarget,
		Subscribed:      m.subscribed,
		Nonce:           m.nonce,
		Agent:           m.agent,
		Username:        m.username,
		Minerid:         m.minerid,
		Authorized:      m.authorized,
	}
}
