package stratum_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	. "github.com/FactomWyomingEntity/prosper-pool/stratum"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func serverAndClient(t *testing.T) (s *Server, miner *Client, srv net.Conn, cli net.Conn) {
	require := require.New(t)

	s, err := NewServer(viper.GetViper())
	require.NoError(err)

	srv, cli = net.Pipe()
	miner, err = NewClient("user", "miner", "password", "invitecode", "payoutaddress", "0.0.1")
	require.NoError(err)

	miner.InitConn(cli)
	s.NewConn(srv)
	return s, miner, srv, cli
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

func TestServer_GetVersion(t *testing.T) {
	require := require.New(t)
	s, m, _, _ := serverAndClient(t)
	for s.Miners.Len() == 0 { // Wait for miner to be added
		time.Sleep(20 * time.Millisecond)
	}
	ctx := context.Background()
	go m.Listen(ctx)

	for _, k := range s.Miners.ListMiners() {
		err := s.GetVersion(k)
		require.NoError(err)
	}
}

func TestServer_ReconnectClient(t *testing.T) {
	require := require.New(t)
	srv, miner, _, cli := serverAndClient(t)

	err := miner.Subscribe()
	require.NoError(err)

	ctx := context.Background()
	go miner.Listen(ctx)

	r := bufio.NewReader(cli)
	_, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	initialLength := len(srv.Miners.ListMiners())
	require.Greater(initialLength, 0)

	initialMinerSessionID := srv.Miners.ListMiners()[0]
	newViper := viper.GetViper()
	newViper.Set("stratum.StratumPort", 1234)
	srvr, err := NewServer(newViper)
	require.NoError(err)

	go srvr.Listen(ctx)
	err = srv.ReconnectClient(initialMinerSessionID, "0.0.0.0", "1234", "2")

	time.Sleep(1 * time.Second)
	// Client should be disconnected by now
	time.Sleep(2 * time.Second)
	newLength := len(srvr.Miners.ListMiners())
	require.Greater(newLength, 0)
	newMinerSessionID := srvr.Miners.ListMiners()[0]

	require.NotEqual(initialMinerSessionID, newMinerSessionID)
	require.NoError(err)
}

func TestServer_ShowMessage(t *testing.T) {
	require := require.New(t)
	s, m, _, _ := serverAndClient(t)
	for s.Miners.Len() == 0 { // Wait for miner to be added
		time.Sleep(20 * time.Millisecond)
	}

	ctx := context.Background()
	go m.Listen(ctx)

	for _, k := range s.Miners.ListMiners() {
		err := s.ShowMessage(k, "Test message")
		require.NoError(err)
		// TODO: actually ensure message is printed/logged client-side
	}
}

func TestServer_SetNonce(t *testing.T) {
	require := require.New(t)
	srv, miner, _, cli := serverAndClient(t)

	err := miner.Subscribe()
	require.NoError(err)

	ctx := context.Background()
	go miner.Listen(ctx)

	r := bufio.NewReader(cli)
	_, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	err = srv.SetNonce(srv.Miners.ListMiners()[0], "ffeabea") // 268348394 in decimal
	require.NoError(err)
	// TODO: ensure client miner has updated nonce internally (once this is being done)
}

func TestServer_SetTarget(t *testing.T) {
	require := require.New(t)
	srv, miner, _, cli := serverAndClient(t)

	err := miner.Subscribe()
	require.NoError(err)

	ctx := context.Background()
	go miner.Listen(ctx)

	r := bufio.NewReader(cli)
	_, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	err = srv.SetTarget(srv.Miners.ListMiners()[0], "ffeabea") // 268348394 in decimal
	require.NoError(err)
	// TODO: ensure client miner has updated target internally (once this is being done)
}
