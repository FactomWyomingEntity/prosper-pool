package stratum

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Server struct {
	config *viper.Viper
}

func NewServer(conf *viper.Viper) (*Server, error) {
	s := new(Server)
	s.config = conf
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

		// TODO: Add miners to an array or map in a thread safe manner.
		//		We will need to broadcast to all miners for new jobs.
		m := Miner{conn: conn}
		go s.handleClient(&m)
	}
}

type Miner struct {
	log  *log.Entry
	conn *net.TCPConn
	// TODO: Manage all miner state. Like authentication, jobs, shares, etc
}

func (s Server) handleClient(client *Miner) {
	client.log = log.WithFields(log.Fields{"ip": client.conn.RemoteAddr().String()})
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

		s.handleMessage(client, data)
	}
}

func (s Server) handleMessage(client *Miner, data []byte) {
	var u UnknownRPC
	err := json.Unmarshal(data, &u)
	if err != nil {
		client.log.WithError(err).Warnf("client read failed")
	}

	if u.IsRequest() {
		req := u.GetRequest()
		// TODO: Handle req
		var _ = req
	} else {
		resp := u.GetResponse()
		// TODO: Handle resp
		var _ = resp
	}

	// TODO: Don't just print everything
	client.log.Infof(string(data))
}
