package stratum_test

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"

	"github.com/FactomWyomingEntity/private-pool/stratum"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestServer_Subscribe(t *testing.T) {
	require := require.New(t)
	// TODO: Replace config with some defaults
	s, err := stratum.NewServer(viper.GetViper())
	require.NoError(err)

	srv, cli := net.Pipe()
	miner, err := stratum.NewClient()
	require.NoError(err)

	miner.InitConn(cli)
	s.NewConn(srv)

	err = miner.Subscribe()
	require.NoError(err)

	r := bufio.NewReader(cli)
	data, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	var resp stratum.Response
	err = json.Unmarshal(data, &resp)
	require.NoError(err)
	require.NotZero(resp.ID)
	require.Nil(resp.Error)

	// Check the response
	var sRes stratum.SubscribeResult
	err = resp.FitResult(&sRes)
	require.NoError(err)

	if len(sRes) != 2 {
		t.Error("exp string array of 2")
	}
}
