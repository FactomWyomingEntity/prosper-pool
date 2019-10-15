package stratum

import (
	"bufio"
	"encoding/json"
	"net"

	log "github.com/sirupsen/logrus"
)

var _ = log.Println

// Clients talk to stratum servers. They are on the miner side of things, so their config's
// should be extremely light, if any.
type Client struct {
	enc  *json.Encoder
	dec  *bufio.Reader
	conn net.Conn
}

func NewClient() (*Client, error) {
	s := new(Client)
	return s, nil
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

	// Wait for subscribe response

	err = c.Authorize("user", "password")
	if err != nil {
		return err
	}

	return nil
}

// JustConnect will not start the handshake process. Good for unit tests
func (c *Client) InitConn(conn net.Conn) {
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = bufio.NewReader(conn)
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
func (c Client) Submit(username, jobID, nonce, oprHash string) error {
	err := c.enc.Encode(SubmitRequest(username, jobID, nonce, oprHash))
	if err != nil {
		return err
	}
	return nil
}

// Subscribe to stratum pool
func (c Client) Subscribe() error {
	err := c.enc.Encode(SubscribeRequest())
	if err != nil {
		return err
	}
	return nil
}

// Suggest preferred mining difficulty to server
func (c Client) SuggestDifficulty(preferredDifficulty string) error {
	err := c.enc.Encode(SuggestDifficultyRequest(preferredDifficulty))
	if err != nil {
		return err
	}
	return nil
}
