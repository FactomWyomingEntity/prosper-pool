package stratum_test

import (
	"bufio"
	"encoding/json"
	"strings"
	"testing"
	"time"

	. "github.com/FactomWyomingEntity/prosper-pool/stratum"
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

	err = miner.Authorize("user,miner", "password", "invitecode", "payoutaddress")
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

func TestClient_GetOPRHash(t *testing.T) {
	require := require.New(t)
	_, miner, _, cli := serverAndClient(t)

	err := miner.Subscribe()
	require.NoError(err)

	r := bufio.NewReader(cli)
	data, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	err = miner.GetOPRHash("exampleJobId")
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
	var oprHashResponse string
	err = resp.FitResult(&oprHashResponse)

	require.NoError(err)
	require.NotEmpty(oprHashResponse)
	require.EqualValues(len(oprHashResponse), 64)
}

func TestClient_Submit(t *testing.T) {
	require := require.New(t)
	_, miner, _, cli := serverAndClient(t)

	err := miner.Subscribe()
	require.NoError(err)

	r := bufio.NewReader(cli)
	data, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	err = miner.Authorize("user,miner", "password", "invitecode", "payoutaddress")
	require.NoError(err)

	data, isPrefix, err = r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	err = miner.Submit("user", "exampleJobId", "a34bdeadbeef892", "00037f39cf870a1f49129f9c82d935665d352ffd25ea3296208f6f7b16fd654f", "ffffffffffffffff49129f9c82d935665d352ffd25ea3296208f6f7b16fd654f")
	require.NoError(err)

	// TODO: actually test that PoW hash meets target, jobId exists, etc

	data, isPrefix, err = r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	var resp Response
	err = json.Unmarshal(data, &resp)
	require.NoError(err)
	require.NotZero(resp.ID)
	require.Nil(resp.Error)

	// Check the response
	var submitRes bool
	err = resp.FitResult(&submitRes)
	require.NoError(err)

	require.True(submitRes)
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

func TestClient_SuggestTarget(t *testing.T) {
	require := require.New(t)
	srv, miner, _, cli := serverAndClient(t)

	err := miner.Subscribe()
	require.NoError(err)

	r := bufio.NewReader(cli)
	_, isPrefix, err := r.ReadLine()
	require.NoError(err)
	require.False(isPrefix)

	err = miner.SuggestTarget("ffeabea") // 268348394 in decimal
	require.NoError(err)
	actualMiner, err := srv.Miners.GetMiner(srv.Miners.ListMiners()[0])
	require.NoError(err)
	time.Sleep(1 * time.Second)
	require.True(strings.Contains(actualMiner.ToString(), "Preferred Target: 268348394"))
}
