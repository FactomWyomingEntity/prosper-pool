package stratum

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FactomWyomingEntity/prosper-pool/authentication"
	"github.com/FactomWyomingEntity/prosper-pool/config"
	"github.com/FactomWyomingEntity/prosper-pool/difficulty"
	"github.com/pegnet/pegnet/modules/opr"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Server struct {
	// miners is a map of miners to their session id
	Miners     *MinerMap
	config     *viper.Viper
	currentJob *Job

	// For any user authentication
	Auth *authentication.Authenticator

	// Will assist in rejecting stale shares
	ShareGate ShareCheck

	configuration struct {
		RequireAuth    bool // Require actual username from miners
		ValidateShares bool
	}

	// We forward submissions to any listeners
	submissionExports []chan<- *ShareSubmission

	stratumPort    int
	welcomeMessage string
}

type ShareSubmission struct {
	Username string `json:"username,omitempty"`
	MinerID  string `json:"minerid,omitempty"`
	JobID    int32  `gorm:"index:jobid" json:"jobid"`
	OPRHash  []byte `json:"oprhash,omitempty"` // Bytes to ensure valid oprhash
	Nonce    []byte `json:"nonce,omitempty"`   // Bytes to ensure valid nonce
	Target   uint64 `json:"target,omitempty"`  // Uint64 to ensure valid target
}

type Job struct {
	JobID   int32         `json:"jobid"`
	OPRHash string        `json:"oprhash"`
	OPR     opr.V2Content // This will be deprecated
	OPRv4   opr.V4Content
}

func (j Job) JobIDString() string {
	return fmt.Sprintf("%d", j.JobID)
}

// JobIDFromHeight is just a standard function to get the jobid for a height.
// If we decide to extend the jobids, we can more easily control it with a
// function.
func JobIDFromHeight(height int32) int32 {
	return height
}

func NewServer(conf *viper.Viper) (*Server, error) {
	s := new(Server)
	s.config = conf
	s.Miners = NewMinerMap()

	s.configuration.RequireAuth = conf.GetBool(config.ConfigStratumRequireAuth)
	// Stub this out so we don't get a nil dereference
	s.ShareGate = new(AlwaysYesShareCheck)
	s.stratumPort = conf.GetInt(config.ConfigStratumPort)
	s.welcomeMessage = conf.GetString(config.ConfigStratumWelcomeMessage)
	s.configuration.ValidateShares = conf.GetBool(config.ConfigStratumCheckAllWork)
	if s.configuration.ValidateShares {
		InitLX()
	}

	return s, nil
}

func (s *Server) SetShareCheck(sc ShareCheck) {
	s.ShareGate = sc
}

func (s *Server) SetAuthenticator(auth *authentication.Authenticator) {
	s.Auth = auth
}

// UpdateCurrentJob sets currently-active job details on the stratum server
// and automatically pushes a notification to all connected miners
func (s *Server) UpdateCurrentJob(job *Job) {
	s.currentJob = job
	s.Notify(job)
}

// Notify will notify all miners of a new block to mine
func (s *Server) Notify(job *Job) {
	jobReq := NotifyRequest(job.JobIDString(), job.OPRHash, "")
	data, _ := json.Marshal(jobReq)
	errs := s.Miners.Notify(json.RawMessage(data))
	var _ = errs
	// TODO: handle errs
}

func (s *Server) Listen(ctx context.Context) {
	host := fmt.Sprintf("0.0.0.0:%d", s.stratumPort)
	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		log.WithError(err).Fatal("failed to launch stratum server")
	}

	server, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.WithError(err).Fatal("failed to launch stratum server")
	}

	// Capture a cancel and close the server
	go func() {
		select {
		case <-ctx.Done():
			log.Infof("closing stratum server")
			_ = server.Close()
			return
		}
	}()

	defer server.Close()

	log.Printf("Stratum server listening on %s", addr)

	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			if ctx.Err() != nil {
				return // Server was closed
			}
			continue
		}
		_ = conn.SetKeepAlive(true)
		s.NewConn(conn)
	}
}

// NewConn handles all new conns from the listen. By factoring this out, we
// can create unit tests using net.Pipe()
func (s *Server) NewConn(conn net.Conn) {
	m := InitMiner(conn)
	go s.HandleClient(m)
	go s.HandleBroadcasts(m)
}

type Miner struct {
	log     *log.Entry
	conn    net.Conn
	enc     *json.Encoder
	encSync sync.Mutex // All encodes should be synchronized
	// TODO: Manage all miner state. Like authentication, jobs, shares, etc

	preferredTarget uint64

	// broadcast will broadcast any notify messages to this miner
	broadcast chan interface{}

	// State information
	subscribed bool
	sessionID  string
	ip         string
	nonce      uint32
	agent      string // Agent/version from subscribe
	username   string
	minerid    string
	authorized bool

	joined time.Time

	// nonceHistory is used to prevent miners from submitting the same
	// nonce for the same job.
	nonceHistory map[string]struct{}
	nonceLock    sync.RWMutex
}

// InitMiner starts a new miner with the needed encoders and channels set up
func InitMiner(conn net.Conn) *Miner {
	m := new(Miner)
	m.ip = conn.RemoteAddr().String()
	m.conn = conn
	m.enc = json.NewEncoder(conn)
	m.log = log.WithFields(log.Fields{"ip": m.conn.RemoteAddr().String()})
	// To push the encoding time to the individual threads, rather than
	// the looping over all miners
	m.broadcast = make(chan interface{}, 2)
	m.nonceHistory = make(map[string]struct{})

	return m
}

func (m *Miner) NewNonce(nonce string) bool {
	m.nonceLock.Lock()
	_, ok := m.nonceHistory[nonce]
	if !ok {
		m.nonceHistory[nonce] = struct{}{}
	}
	m.nonceLock.Unlock()
	return ok
}

func (m *Miner) ResetNonceHistory() {
	m.nonceLock.Lock()
	m.nonceHistory = make(map[string]struct{})
	m.nonceLock.Unlock()
}

// Close shuts down miner's broadcast channel
func (m *Miner) Close() {
	close(m.broadcast)
}

// ToString returns a string representation of the internal miner client state
func (m *Miner) ToString() string {
	return fmt.Sprintf("Session ID: %s\nIP: %s\nAgent: %s\nPreferred Target: %d\nSubscribed: %t\nAuthorized: %t\nNonce: %d", m.sessionID, m.conn.RemoteAddr().String(), m.agent, m.preferredTarget, m.subscribed, m.authorized, m.nonce)
}

// Broadcast should accept the already json marshalled msg
func (m *Miner) Broadcast(msg json.RawMessage) (err error) {
	defer func() {
		// This should never happen, but we don't want a bugged miner taking us
		// down.
		if r := recover(); r != nil {
			err = fmt.Errorf("miner was closed")
		}
	}()
	select {
	case m.broadcast <- msg:
		return nil
	default:
		return fmt.Errorf("channel full")
	}
}

// HandleBroadcasts will send out all pool broadcast messages to the miners.
// This handles all notify messages
func (s *Server) HandleBroadcasts(client *Miner) {
	for {
		select {
		case msg, ok := <-client.broadcast:
			if !ok {
				return
			}
			client.encSync.Lock()
			err := client.enc.Encode(msg)
			client.encSync.Unlock()
			if err == io.EOF {
				client.log.Infof("client disconnected")
				return
			}
			if err != nil {
				client.log.WithError(err).Warn("failed to notify")
			}
		}
	}
}

func (s *Server) HandleClient(client *Miner) {
	// Register this new miner
	s.Miners.AddMiner(client)
	defer s.Miners.DisconnectMiner(client)

	reader := bufio.NewReader(client.conn)
	for {
		data, isPrefix, err := reader.ReadLine()
		if isPrefix {
			// TODO: We can increase the buffer in the NewReader
			client.log.Warnf("too many bytes on pipe")
			break
		} else if err == io.EOF {
			client.log.Infof("client disconnected")
			break
		} else if err != nil {
			client.log.WithError(err).Warnf("client read failed")
			break
		}

		client.encSync.Lock()
		s.HandleMessage(client, data)
		client.encSync.Unlock()
	}
}

func (s *Server) HandleMessage(client *Miner, data []byte) {
	var u UnknownRPC
	err := json.Unmarshal(data, &u)
	if err != nil {
		client.log.WithError(err).Warnf("client read failed")
	}

	if u.IsRequest() {
		req := u.GetRequest()
		s.HandleRequest(client, req)
	} else {
		resp := u.GetResponse()
		// TODO: Handle resp
		var _ = resp
	}

	//client.log.Infof(string(data))
}

func (s *Server) HandleRequest(client *Miner, req Request) {
	var params RPCParams
	if err := req.FitParams(&params); err != nil {
		client.log.WithField("method", req.Method).Warnf("bad params %s", req.Method)
		_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
		return
	}

	switch req.Method {
	case "mining.authorize":
		// "params": ["username,minerid", "password", "invitecode", "payoutaddress"]
		if len(params) < 1 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}
		// Ignore the session id if provided in the params
		arr := strings.Split(params[0], ",")
		if len(arr) != 2 {
			_ = client.enc.Encode(HelpfulRPCError(req.ID, ErrorInvalidParams, "authorize requires 'username,minerid'"))
			return
		}

		client.username = arr[0]
		client.minerid = arr[1]
		client.log = client.log.WithFields(log.Fields{"minerid": client.minerid, "username": client.username})

		if s.Auth != nil && s.configuration.RequireAuth {
			if !s.Auth.Exists(client.username) {
				// Did they provide a password, code, and payout addr?
				if len(params) >= 4 && s.Auth.RegisterUser(client.username, params[1], params[2], params[3]) {
					// User registered! Let them through by falling out of this if statement
				} else {
					// User rejected
					// TODO: Provide a reason?
					// TODO: Disconnect them?
					if err := client.enc.Encode(AuthorizeResponse(req.ID, false, nil)); err != nil {
						client.log.WithField("method", req.Method).WithError(err).Error("failed to send message")
					}
					return
				}
			}
		}

		if err := client.enc.Encode(AuthorizeResponse(req.ID, true, nil)); err != nil {
			client.log.WithField("method", req.Method).WithError(err).Error("failed to send message")
		} else {
			client.authorized = true
			s.ShowMessage(client.sessionID, s.welcomeMessage)
		}
	case "mining.get_oprhash":
		if len(params) < 1 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}

		// TODO: actually retrieve OPR hash for the given jobID (for now using dummy data)
		dummyOPRHash := "00011111af870a1f49129f9c82d935665d352fffffea3296208f6f7b16faaabc"

		if err := client.enc.Encode(GetOPRHashResponse(req.ID, dummyOPRHash)); err != nil {
			client.log.WithField("method", req.Method).WithError(err).Error("failed to send message")
		}
	case "mining.submit":
		// "params": ["username", "jobID", "nonce", "oprHash", "target"]
		if len(params) < 5 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}

		if params[0] != client.username {
			_ = client.enc.Encode(HelpfulRPCError(req.ID, ErrorInvalidParams, "username not as expected"))
			return
		}

		if !s.ProcessSubmission(client, params[1], params[2], params[3], params[4]) {
			// Rejected share
			// ignore errors on reject shares
			_ = client.enc.Encode(SubmitResponse(req.ID, false, nil))
			return
		}

		if err := client.enc.Encode(SubmitResponse(req.ID, true, nil)); err != nil {
			client.log.WithField("method", req.Method).WithError(err).Error("failed to send message")
		}
	case "mining.subscribe":
		if len(params) < 1 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}
		// Ignore the session id if provided in the params
		client.agent = params[0]

		if err := client.enc.Encode(SubscribeResponse(req.ID, client.sessionID, client.nonce)); err != nil {
			client.log.WithField("method", req.Method).WithError(err).Error("failed to send message")
		} else {
			client.subscribed = true

			// TODO: Use vardiff
			client.preferredTarget = difficulty.PDiff
			err = s.SetTarget(client.sessionID, fmt.Sprintf("%x", difficulty.PDiff))
			if err != nil {
				log.WithError(err).Error("failed to set target")
			}
			// Notify newly-subscribed client with current job details
			if s.currentJob != nil {
				err = s.SingleClientNotify(client.sessionID, s.currentJob.JobIDString(), s.currentJob.OPRHash, "")
				if err != nil {
					log.WithError(err).Error("failed to send job")
				}
			}
		}
	case "mining.suggest_target":
		if len(params) < 1 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}

		// We will not accept suggested targets from miners at this time
		//preferredTarget, err := strconv.ParseUint(params[0], 16, 64)
		//if err == nil {
		//	client.preferredTarget = preferredTarget
		//}
	default:
		client.log.Warnf("unknown method %s", req.Method)
		_ = client.enc.Encode(QuickRPCError(req.ID, ErrorMethodNotFound))
	}
}

// ProcessSubmission will forward the shares and return if the share was accepted
func (s *Server) ProcessSubmission(miner *Miner, jobID, nonce, oprHash, target string) bool {
	sLog := log.WithFields(log.Fields{"user": miner.username, "miner": miner.minerid, "job": jobID})
	if s.currentJob == nil {
		return false // No current job
	}

	if jobID != s.currentJob.JobIDString() || oprHash != s.currentJob.OPRHash {
		return false // Only accepts current job
	}

	// Double check the fields
	oB, err := hex.DecodeString(oprHash)
	if err != nil {
		sLog.WithError(err).Errorf("miner provided bad oprhash")
		return false
	}

	nB, err := hex.DecodeString(nonce)
	if err != nil {
		sLog.WithError(err).Errorf("miner provided bad nonce")
		return false
	}

	tU, err := strconv.ParseUint(target, 16, 64)
	if err != nil {
		sLog.WithError(err).Errorf("miner provided bad target")
		return false
	}

	if tU < miner.preferredTarget {
		return false
	}

	// Check if this is a duplicate nonce
	if miner.NewNonce(nonce) {
		return false
	}

	jobHeight, err := strconv.ParseInt(jobID, 10, 32)
	if err != nil {
		sLog.WithError(err).Errorf("miner provided bad jobid")
		return false
	}

	if s.configuration.ValidateShares {
		if !Validate(oB, nB, tU) {
			return false // Submitted a bad share
		}
	}

	// Check if we can accept shares right now
	// E.g: If we are between minute 0 and minute 1, the job is
	// stale
	if !s.ShareGate.CanSubmit() {
		return false
	}

	submit := &ShareSubmission{
		Username: miner.username,
		MinerID:  miner.minerid,
		JobID:    int32(jobHeight),
		OPRHash:  oB,
		Nonce:    nB,
		Target:   tU,
	}

	for _, export := range s.submissionExports {
		select { // Non blocking
		case export <- submit:
		default:
			sLog.Warnf("failed to export share")
		}
	}

	return true
}

func (s *Server) GetVersion(clientName string) error {
	miner, err := s.Miners.GetMiner(clientName)
	if err != nil {
		return err
	}
	err = miner.enc.Encode(GetVersionRequest())
	return err
}

func (s *Server) SingleClientNotify(clientName, jobID, oprHash, cleanjobs string) error {
	miner, err := s.Miners.GetMiner(clientName)
	if err != nil {
		return err
	}
	err = miner.enc.Encode(NotifyRequest(jobID, oprHash, cleanjobs))
	return err
}

func (s *Server) ReconnectClient(clientName, hostname, port, waittime string) error {
	miner, err := s.Miners.GetMiner(clientName)
	if err != nil {
		return err
	}
	err = miner.enc.Encode(ReconnectRequest(hostname, port, waittime))
	return err
}

func (s *Server) SetTarget(clientName, target string) error {
	miner, err := s.Miners.GetMiner(clientName)
	if err != nil {
		return err
	}
	err = miner.enc.Encode(SetTargetRequest(target))
	return err
}

func (s *Server) SetNonce(clientName, nonce string) error {
	miner, err := s.Miners.GetMiner(clientName)
	if err != nil {
		return err
	}
	err = miner.enc.Encode(SetNonceRequest(nonce))
	return err
}

func (s *Server) ShowMessage(clientName, message string) error {
	miner, err := s.Miners.GetMiner(clientName)
	if err != nil {
		return err
	}
	err = miner.enc.Encode(ShowMessageRequest(message))
	return err
}

func (s *Server) StopMining(clientName string) error {
	miner, err := s.Miners.GetMiner(clientName)
	if err != nil {
		return err
	}
	err = miner.enc.Encode(StopMiningRequest())
	return err
}

// GetSubmissionExport should be called in an init phase, so does not
// need to be thread safe
func (s *Server) GetSubmissionExport() <-chan *ShareSubmission {
	c := make(chan *ShareSubmission, 250)
	s.submissionExports = append(s.submissionExports, c)
	return c
}
