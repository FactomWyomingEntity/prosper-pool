package stratum_test

import (
	"encoding/json"
	"testing"

	"github.com/FactomWyomingEntity/prosper-pool/stratum"
)

func TestUnknownRPC_IsRequest(t *testing.T) {
	var u stratum.UnknownRPC
	j := `{"id": 1, "method": "mining.subscribe", "params": ["MyMiner/1.0.0", null, "my.pool.com", 1234]}`
	err := json.Unmarshal([]byte(j), &u)
	if err != nil {
		t.Error(err)
	}
	if !u.IsRequest() {
		t.Error("exp request")
	}
}
