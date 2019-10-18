package stratum

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

var _ = log.Println

// Clients talk to stratum servers. They are on the miner side of things, so their config's
// should be extremely light, if any.
type Client struct {
	enc  *json.Encoder
	dec  *bufio.Reader
	conn net.Conn

	version string

	subscriptions []Subscription
	verbose       bool
}

func NewClient(verbose bool) (*Client, error) {
	c := new(Client)
	c.verbose = verbose
	c.version = "0.0.1"
	return c, nil
}

func (c *Client) Connect(address string) error {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}

	return c.Handshake(conn)
}

func (c *Client) Handshake(conn net.Conn) error {
	c.InitConn(conn)
	err := c.Subscribe()
	if err != nil {
		return err
	}

	// Receive subscribe response
	data, _, err := c.dec.ReadLine()
	var resp Response
	err = json.Unmarshal(data, &resp)

	if c.verbose {
		log.Printf("CLIENT READ: %s\n", string(data))
	}

	err = c.Authorize("user", "password")
	if err != nil {
		return err
	}

	data, _, err = c.dec.ReadLine()
	err = json.Unmarshal(data, &resp)
	if c.verbose {
		log.Printf("CLIENT READ: %s\n", string(data))
	}
	return nil
}

// JustConnect will not start the handshake process. Good for unit tests
func (c *Client) InitConn(conn net.Conn) {
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = bufio.NewReader(conn)
}

// Wait waittime seconds, then proceed with Connect
func (c *Client) WaitThenConnect(address, waittime string) error {
	i, err := strconv.ParseInt(waittime, 10, 64)
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(i) * time.Second)
	return c.Connect(address)
}

// Authorize against stratum pool
func (c Client) Authorize(username, password string) error {
	err := c.enc.Encode(AuthorizeRequest(username, password))
	if err != nil {
		return err
	}
	return nil
}

// Request current OPR hash from server
func (c Client) GetOPRHash(jobID string) error {
	err := c.enc.Encode(GetOPRHashRequest(jobID))
	if err != nil {
		return err
	}
	return nil
}

// Submit completed work to server
func (c Client) Submit(username, jobID, nonce, oprHash, target string) error {
	err := c.enc.Encode(SubmitRequest(username, jobID, nonce, oprHash, target))
	if err != nil {
		return err
	}
	return nil
}

// Subscribe to stratum pool
func (c Client) Subscribe() error {
	err := c.enc.Encode(SubscribeRequest(c.version))
	if err != nil {
		return err
	}
	return nil
}

// Suggest preferred mining target to server
func (c Client) SuggestTarget(preferredTarget string) error {
	err := c.enc.Encode(SuggestTargetRequest(preferredTarget))
	if err != nil {
		return err
	}
	return nil
}

func (c Client) Listen(ctx context.Context) {
	defer c.conn.Close()
	// Capture a cancel and close the server
	go func() {
		select {
		case <-ctx.Done():
			log.Infof("shutting down stratum client")
			c.conn.Close()
			return
		}
	}()

	log.Printf("Stratum client listening to server at %s\n", c.conn.RemoteAddr().String())

	r := bufio.NewReader(c.conn)

	for {
		readBytes, _, err := r.ReadLine()
		if err != nil {
			return
		} else {
			c.HandleMessage(readBytes)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c Client) HandleMessage(data []byte) {
	var u UnknownRPC
	err := json.Unmarshal(data, &u)
	if err != nil {
		log.WithError(err).Warnf("client read failed")
	}

	if u.IsRequest() {
		req := u.GetRequest()
		c.HandleRequest(req)
	} else {
		resp := u.GetResponse()
		// TODO: Handle resp
		var _ = resp
	}

	// TODO: Don't just print everything
	log.Infof(string(data))
}

func (c Client) HandleRequest(req Request) {
	var params RPCParams
	if err := req.FitParams(&params); err != nil {
		log.WithField("method", req.Method).Warnf("bad params %s", req.Method)
		return
	}

	switch req.Method {
	case "client.get_version":
		if err := c.enc.Encode(GetVersionResponse(req.ID, c.version)); err != nil {
			log.WithField("method", req.Method).WithError(err).Error("failed to respond to get_version")
		}
	case "client.reconnect":
		if len(params) < 2 {
			log.Errorf("Not enough parameters to reconnect with: %s\n", params)
			return
		}

		waittime := "0"
		if len(params) > 2 {
			_, err := strconv.ParseInt(params[2], 10, 64)
			if err == nil {
				waittime = params[2]
			}
		}

		if err := c.WaitThenConnect(params[0]+":"+params[1], waittime); err != nil {
			log.WithField("method", req.Method).WithError(err).Error("failed to reconnect")
		}
	case "client.show_message":
		if len(params) < 1 {
			log.Errorln("No message to show")
			return
		}
		// Print & log message in human-readable way
		fmt.Printf("\n\nMessage from server: %s\n\n\n", params[0])
		log.Printf("Message from server: %s\n", params[0])
	case "mining.notify":
		if len(params) < 2 {
			log.Errorf("Not enough parameters from notify: %s\n", params)
			return
		}

		jobID := params[0]
		oprHash := params[1]

		log.Printf("JobID: %s ... OPR Hash: %s\n", jobID, oprHash)
		// TODO: do more than just log the notification details (actually update miner)
	case "mining.set_target":
		if len(params) < 1 {
			log.Errorf("Not enough parameters from set_target: %s\n", params)
			return
		}

		newTarget := params[0]

		log.Printf("New Target: %s\n", newTarget)
		// TODO: do more than just log the newTarget details (actually update miner)
	case "mining.set_nonce":
		if len(params) < 1 {
			log.Errorf("Not enough parameters from set_nonce: %s\n", params)
			return
		}

		nonce := params[0]

		log.Printf("New Nonce: %s\n", nonce)
		// TODO: do more than just log the nonce details (actually update miner job)
	default:
		log.Warnf("unknown method %s", req.Method)
	}
}
