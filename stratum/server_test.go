package stratum_test

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
	"time"

	. "github.com/FactomWyomingEntity/private-pool/stratum"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func serverAndClient(t *testing.T) (s *Server, miner *Client, srv net.Conn, cli net.Conn) {
	require := require.New(t)

	// TODO: Replace config with some defaults
	s, err := NewServer(viper.GetViper())
	require.NoError(err)

	srv, cli = net.Pipe()
	miner, err = NewClient()
	require.NoError(err)

	miner.InitConn(cli)
	s.NewConn(srv)
	return s, miner, srv, cli
}

func TestServer_Subscribe(t *testing.T) {
	require := require.New(t)
	_, miner, _, cli := serverAndClient(t)

	err := miner.Subscribe()
	require.NoError(err)

	r := bufio.NewReader(cli)
	data, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	var resp Response
	err = json.Unmarshal(data, &resp)
	require.NoError(err)
	require.NotZero(resp.ID)
	require.Nil(resp.Error)

	// Check the response
	var sRes SubscribeResult
	err = resp.FitResult(&sRes)
	require.NoError(err)

	if len(sRes) != 2 {
		t.Error("exp string array of 2")
	}
}

func TestServer_Notify(t *testing.T) {
	require := require.New(t)
	s, _, _, cli := serverAndClient(t)
	for s.Miners.Len() == 0 { // Wait for miner to be added
		time.Sleep(20 * time.Millisecond)
	}

	exp, _ := json.Marshal("test")
	errs := s.Miners.Notify(exp)
	require.Zero(len(errs))

	r := bufio.NewReader(cli)
	data, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)
	if string(data) != string(exp) {
		t.Errorf("exp '%s' got '%s'", string(exp), string(data))
	}
}
