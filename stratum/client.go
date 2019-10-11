package stratum

import (
	"encoding/json"
	"net"

	log "github.com/sirupsen/logrus"
)

var _ = log.Println

// Clients talk to stratum servers. They are on the miner side of things, so their config's
// should be extremely light, if any.
type Client struct {
	enc  *json.Encoder
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

	return c.ConnectFromConn(conn)
}

func (c *Client) ConnectFromConn(conn net.Conn) error {
	c.InitConn(conn)

	err := c.Subscribe()
	if err != nil {
		return err
	}

	return nil
}

// JustConnect will not start the handshake process. Good for unit tests
func (c *Client) InitConn(conn net.Conn) {
	c.conn = conn
	c.enc = json.NewEncoder(conn)
}

// Subscribe to stratum pool
func (c Client) Subscribe() error {
	err := c.enc.Encode(SubscribeRequest())
	if err != nil {
		return err
	}
	return nil
}
