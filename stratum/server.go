package stratum

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Server struct {
	// miners is a map of miners to their session id
	Miners *MinerMap
	config *viper.Viper
}

func NewServer(conf *viper.Viper) (*Server, error) {
	s := new(Server)
	s.config = conf
	s.Miners = NewMinerMap()
	return s, nil
}

func (s Server) Listen(ctx context.Context) {
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
func (s Server) NewConn(conn net.Conn) {
	m := InitMiner(conn)
	go s.HandleClient(m)
}

type Miner struct {
	log  *log.Entry
	conn net.Conn
	enc  *json.Encoder
	// TODO: Manage all miner state. Like authentication, jobs, shares, etc

	// State information
	subscribed  bool
	sessionID   string
	extraNonce1 uint32
	agent       string // Agent/version from subscribe

	joined time.Time
}

func InitMiner(conn net.Conn) *Miner {
	m := new(Miner)
	m.conn = conn
	m.enc = json.NewEncoder(conn)
	m.log = log.WithFields(log.Fields{"ip": m.conn.RemoteAddr().String()})

	return m
}

func (s Server) HandleClient(client *Miner) {
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

		s.HandleMessage(client, data)
	}
}

func (s Server) HandleMessage(client *Miner, data []byte) {
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

func (s Server) HandleRequest(client *Miner, req Request) {
	switch req.Method {
	case "mining.subscribe":
		var params SubscribeParams
		if err := req.FitParams(&params); err != nil {
			client.log.WithField("method", req.Method).Warnf("bad params %s", req.Method)
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}

		if len(params) < 1 {
			_ = client.enc.Encode(QuickRPCError(req.ID, ErrorInvalidParams))
			return
		}
		// Ignore the session id if provided in the params
		client.agent = params[0]

		if err := client.enc.Encode(SubscribeResponse(req.ID, client.sessionID, client.extraNonce1)); err != nil {
			client.log.WithField("method", req.Method).WithError(err).Error("failed to send message")
		}
	default:
		client.log.Warnf("unknown method %s", req.Method)
		_ = client.enc.Encode(QuickRPCError(req.ID, ErrorMethodNotFound))
	}
}
