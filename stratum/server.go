package stratum

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/pegnet/pegnet/modules/opr"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Server struct {
	// miners is a map of miners to their session id
	Miners     *MinerMap
	config     *viper.Viper
	currentJob *Job
}

type Job struct {
	JobID   string `json:"jobid"`
	OPRHash string `json:"oprhash"`
	OPR     opr.V2Content
}

// JobIDFromHeight is just a standard function to get the jobid for a height.
// If we decide to extend the jobids, we can more easily control it with a
// function.
func JobIDFromHeight(height int32) string {
	return fmt.Sprintf("%d", height)
}

func NewServer(conf *viper.Viper) (*Server, error) {
	s := new(Server)
	s.config = conf
	s.Miners = NewMinerMap()
	return s, nil
}

// UpdateCurrentJob sets currently-active job details on the stratum server
// and automatically pushes a notification to all connected miners
func (s *Server) UpdateCurrentJob(job *Job) {
	s.currentJob = job
	s.Notify(job)
}

// Notify will notify all miners of a new block to mine
func (s *Server) Notify(job *Job) {
	jobReq := NotifyRequest(job.JobID, job.OPRHash, "")
	data, _ := json.Marshal(jobReq)
	errs := s.Miners.Notify(json.RawMessage(data))
	var _ = errs
	// TODO: handle errs
}

func (s *Server) Listen(ctx context.Context) {
	// TODO: Change this with config file
	host := fmt.Sprintf("0.0.0.0:1234")
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
	nonce      uint32
	agent      string // Agent/version from subscribe
	authorized bool

	joined time.Time
}

// InitMiner starts a new miner with the needed encoders and channels set up
func InitMiner(conn net.Conn) *Miner {
	m := new(Miner)
	m.conn = conn
	m.enc = json.NewEncoder(conn)
	m.log = log.WithFields(log.Fields{"ip": m.conn.RemoteAddr().String()})
	// To push the encoding time to the individual threads, rather than
	// the looping over all miners
	m.broadcast = make(chan interface{}, 2)

	return m
}

// Close shuts down miner's broadcast channel
func (m *Miner) Close() {
	close(m.broadcast)
}

// ToString returns a string representation of the internal miner client state
func (m *Miner) ToString() string {
	return fmt.Sprintf("Session ID: %s\nAgent: %s\nPreferred Target: %d\nSubscribed: %t\nAuthorized: %t\nNonce: %d", m.sessionID, m.agent, m.preferredTarget, m.subscribed, m.authorized, m.nonce)
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

	// TODO: Don't just print everything
	client.log.Infof(string(data))
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
		if len(params) < 1 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}
		// Ignore the session id if provided in the params
		client.agent = params[0]

		// TODO: actually check username/password (if user/pass authentication is desired)
		if err := client.enc.Encode(AuthorizeResponse(req.ID, true, nil)); err != nil {
			client.log.WithField("method", req.Method).WithError(err).Error("failed to send message")
		} else {
			client.authorized = true
		}
	case "mining.get_oprhash":
		if len(params) < 1 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}
		// Ignore the session id if provided in the params
		client.agent = params[0]

		// TODO: actually retrieve OPR hash for the given jobID (for now using dummy data)
		dummyOPRHash := "00037f39cf870a1f49129f9c82d935665d352ffd25ea3296208f6f7b16fd654f"

		if err := client.enc.Encode(GetOPRHashResponse(req.ID, dummyOPRHash)); err != nil {
			client.log.WithField("method", req.Method).WithError(err).Error("failed to send message")
		} else {
			client.authorized = true
		}
	case "mining.submit":
		if len(params) < 1 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}

		// TODO: actually process the submission (ProcessSubmission); for now just pretend success
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

		if err := client.enc.Encode(SubscribeResponse(req.ID, client.sessionID)); err != nil {
			client.log.WithField("method", req.Method).WithError(err).Error("failed to send message")
		} else {
			client.subscribed = true

			// Notify newly-subscribed client with current job details
			if s.currentJob != nil {
				err = s.SingleClientNotify(client.sessionID, s.currentJob.JobID, s.currentJob.OPRHash, "")
				if err != nil {
					log.Error(err)
				}
				// For some reason, double-notifying seems necessary upon initial connection to pool
				err = s.SingleClientNotify(client.sessionID, s.currentJob.JobID, s.currentJob.OPRHash, "")
				if err != nil {
					log.Error(err)
				}
			}
		}
	case "mining.suggest_target":
		if len(params) < 1 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}

		preferredTarget, err := strconv.ParseUint(params[0], 16, 64)
		if err == nil {
			client.preferredTarget = preferredTarget
		}
	default:
		client.log.Warnf("unknown method %s", req.Method)
		_ = client.enc.Encode(QuickRPCError(req.ID, ErrorMethodNotFound))
	}
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
