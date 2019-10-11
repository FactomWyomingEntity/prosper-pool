package stratum

import (
	"net"
)

// Clients talk to stratum servers. They are on the miner side of things, so their config's
// should be extremely light, if any.
type Client struct {
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

	c.conn = conn
	// TODO: All handeshake stuff
	return nil
}
