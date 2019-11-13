package cmd_test

import (
	"testing"

	. "github.com/FactomWyomingEntity/prosper-pool/cmd"
)

func TestAssetListContainsCaseInsensitive(t *testing.T) {
	if !AssetListContainsCaseInsensitive([]string{"a"}, "A") {
		t.Error("Should be equal")
	}
}
