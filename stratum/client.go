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
	requestsMade  map[int]func(Response)
	verbose       bool
}

func NewClient(verbose bool) (*Client, error) {
	c := new(Client)
	c.verbose = verbose
	c.version = "0.0.1"
	c.requestsMade = make(map[int]func(Response))
	return c, nil
}

func (c *Client) Connect(address string) error {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	c.InitConn(conn)
	return err
}

func (c *Client) Handshake() error {
	err := c.Subscribe()
	if err != nil {
		return err
	}

	return c.Authorize("user,miner", "password")
}

// InitConn will not start the handshake process. Good for unit tests
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
	req := AuthorizeRequest(username, password)
	c.requestsMade[req.ID] = func(resp Response) {
		var result bool
		if err := resp.FitResult(&result); err == nil {
			log.Infof("AuthorizeResponse result: %t\n", result)
		}
	}
	err := c.enc.Encode(req)
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
	req := SubscribeRequest(c.version)
	c.requestsMade[req.ID] = func(resp Response) {
		var subscriptions []Subscription
		if err := resp.FitResult(&subscriptions); err == nil {
			log.Println("Subscriptions Results:")
			for _, subscription := range subscriptions {
				log.Println("...", subscription)
			}
		} else {
			log.Error(err)
		}
	}
	err := c.enc.Encode(req)
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

func (c Client) Close() error {
	log.Infof("shutting down stratum client")
	err := c.conn.Close()
	return err
}

func (c Client) Listen(ctx context.Context) {
	defer c.conn.Close()
	// Capture a cancel and close the client
	go func() {
		select {
		case <-ctx.Done():
			c.Close()
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
		c.HandleResponse(resp)
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
	case "mining.stop_mining":
		log.Println("Request to stop mining received")
		// TODO: actually pause mining until new job is received
	default:
		log.Warnf("unknown method %s", req.Method)
	}
}

func (c Client) HandleResponse(resp Response) {
	if val, ok := c.requestsMade[resp.ID]; ok {
		val(resp)
	}
}
