package stratum

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FactomWyomingEntity/private-pool/mining"
	"github.com/pegnet/pegnet/opr"

	log "github.com/sirupsen/logrus"
)

var _ = log.Println

// Clients talk to stratum servers. They are on the miner side of things, so their config's
// should be extremely light, if any.
type Client struct {
	enc  *json.Encoder
	dec  *bufio.Reader
	conn net.Conn

	version        string
	username       string
	minername      string
	currentJobID   string
	currentOPRHash string
	currentTarget  uint64

	miner     *mining.PegnetMiner
	successes chan *mining.Winner

	subscriptions []Subscription
	requestsMade  map[int]func(Response)
	autoreconnect bool
	sync.RWMutex
}

func NewClient(username, minername, password, version string) (*Client, error) {
	c := new(Client)
	c.autoreconnect = true
	c.version = version
	c.username = username
	c.minername = minername
	c.currentJobID = "1"
	c.currentOPRHash = "00037f39cf870a1f49129f9c82d935665d352ffd25ea3296208f6f7b16fd654f"
	c.currentTarget = 0xfffe000000000000
	c.requestsMade = make(map[int]func(Response))

	commandChannel := make(chan *mining.MinerCommand, 10)
	successChannel := make(chan *mining.Winner, 10)
	c.successes = successChannel

	go c.ListenForSuccess()

	opr.InitLX()
	c.miner = mining.NewPegnetMiner(1, commandChannel, successChannel)
	go c.miner.Mine(context.Background())
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

	return c.Authorize(fmt.Sprintf("%s,%s", c.username, c.minername), "password")
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
func (c *Client) Authorize(username, password string) error {
	req := AuthorizeRequest(username, password)
	c.Lock()
	c.requestsMade[req.ID] = func(resp Response) {
		var result bool
		if err := resp.FitResult(&result); err == nil {
			log.Infof("AuthorizeResponse result: %t\n", result)
		}
	}
	c.Unlock()
	err := c.enc.Encode(req)
	if err != nil {
		return err
	}

	return nil
}

// Request current OPR hash from server
func (c *Client) GetOPRHash(jobID string) error {
	req := GetOPRHashRequest(jobID)
	c.Lock()
	c.requestsMade[req.ID] = func(resp Response) {
		var result string
		if err := resp.FitResult(&result); err == nil {
			log.Infof("OPRHash result: %s\n", result)
			if jobID == c.currentJobID {
				newOPRHash, err := hex.DecodeString(result)
				if err != nil {
					log.Error(err)
					return
				}
				command := mining.BuildCommand().
					NewOPRHash(newOPRHash). // New OPR hash to mine
					ResumeMining().         // Start mining
					Build()
				c.miner.SendCommand(command)

			}
		}
	}
	c.Unlock()
	err := c.enc.Encode(req)
	if err != nil {
		return err
	}
	return nil
}

// Submit completed work to server
func (c *Client) Submit(username, jobID, nonce, oprHash, target string) error {
	req := SubmitRequest(username, jobID, nonce, oprHash, target)
	c.Lock()
	c.requestsMade[req.ID] = func(resp Response) {
		var result bool
		if err := resp.FitResult(&result); err == nil {
			log.Infof("Submission result: %t\n", result)
		}
	}
	c.Unlock()
	err := c.enc.Encode(req)
	if err != nil {
		return err
	}
	return nil
}

// Subscribe to stratum pool
func (c *Client) Subscribe() error {
	req := SubscribeRequest(c.version)
	c.Lock()
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
	c.Unlock()
	err := c.enc.Encode(req)
	if err != nil {
		return err
	}
	return nil
}

// Suggest preferred mining target to server
func (c *Client) SuggestTarget(preferredTarget string) error {
	err := c.enc.Encode(SuggestTargetRequest(preferredTarget))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Close() error {
	log.Infof("shutting down stratum client")
	c.autoreconnect = false
	if !reflect.ValueOf(c.conn).IsNil() {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) Listen(ctx context.Context) {
	// Capture a cancel and close the client
	go func() {
		select {
		case <-ctx.Done():
			c.Close()
			return
		}
	}()

	log.Printf("Stratum client listening to server at %s\n", c.conn.RemoteAddr().String())
	originalServerAddress := c.conn.RemoteAddr().String()

	r := bufio.NewReader(c.conn)

	for {
		readBytes, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF || (strings.Contains(err.Error(), "use of closed network connection") && c.autoreconnect) {
				log.Info("Server disconnect detected, attempting reconnect in 5s...")
				if !reflect.ValueOf(c.conn).IsNil() {
					c.conn.Close()
				}
				reconnectError := c.WaitThenConnect(originalServerAddress, "5")
				if reconnectError != nil {
					if strings.Contains(reconnectError.Error(), "connection refused") {
						continue
					} else {
						log.Error(reconnectError)
						return
					}
				} else {
					c.Handshake()
					c.Listen(ctx)
				}
			} else {
				return
			}
		} else {
			c.HandleMessage(readBytes)
		}
	}
}

func (c *Client) HandleMessage(data []byte) {
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

func (c *Client) HandleRequest(req Request) {
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
		newJobID, err := strconv.ParseInt(jobID, 10, 64)
		if err != nil {
			log.Error("Not a valid new JobID")
			return
		}
		existingJobID, _ := strconv.ParseInt(c.currentJobID, 10, 64)
		if newJobID >= existingJobID {
			myHexBytes, err := hex.DecodeString(oprHash)
			if err != nil {
				log.Error(err)
				return
			}
			if newJobID > existingJobID {
				c.currentJobID = jobID
			}
			c.currentOPRHash = oprHash
			command := mining.BuildCommand().
				NewOPRHash(myHexBytes).
				MinimumDifficulty(c.currentTarget).
				ResumeMining().
				Build()
			c.miner.SendCommand(command)
		}
	case "mining.set_target":
		if len(params) < 1 {
			log.Errorf("Not enough parameters from set_target: %s\n", params)
			return
		}

		result, _ := strconv.ParseUint(strings.Replace(params[0], "0x", "", -1), 16, 64)
		c.currentTarget = uint64(result)

		log.Printf("New Target: %x\n", c.currentTarget)

		command := mining.BuildCommand().
			MinimumDifficulty(c.currentTarget).
			Build()
		c.miner.SendCommand(command)
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
		command := mining.BuildCommand().
			PauseMining().
			Build()
		c.miner.SendCommand(command)
	default:
		log.Warnf("unknown method %s", req.Method)
	}
}

func (c *Client) HandleResponse(resp Response) {
	c.Lock()
	if funcToPerform, ok := c.requestsMade[resp.ID]; ok {
		funcToPerform(resp)
		delete(c.requestsMade, resp.ID)
	} else {
		log.Errorf("Response received for unrecognized request ID: %d (ignoring)\n", resp.ID)
	}
	c.Unlock()
}

func (c *Client) ListenForSuccess() {
	for {
		select {
		case winner := <-c.successes:
			c.Submit(c.username, c.currentJobID, winner.Nonce, winner.OPRHash, fmt.Sprintf("%x", c.currentTarget))
		}
	}
}
