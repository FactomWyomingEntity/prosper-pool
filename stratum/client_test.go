package stratum_test

import (
	"bufio"
	"encoding/json"
	"testing"

	. "github.com/FactomWyomingEntity/private-pool/stratum"
	"github.com/stretchr/testify/require"
)

func TestClient_Authorize(t *testing.T) {
	require := require.New(t)
	_, miner, _, cli := serverAndClient(t)

	err := miner.Subscribe()
	require.NoError(err)

	r := bufio.NewReader(cli)
	data, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	err = miner.Authorize("user", "password")
	require.NoError(err)

	data, isPrefix, err = r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	var resp Response
	err = json.Unmarshal(data, &resp)
	require.NoError(err)
	require.NotZero(resp.ID)
	require.Nil(resp.Error)

	// Check the response
	var aRes bool
	err = resp.FitResult(&aRes)
	require.NoError(err)

	require.True(aRes)
}

func TestClient_Subscribe(t *testing.T) {
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
