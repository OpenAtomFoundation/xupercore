package tdpos

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalConfig(t *testing.T) {
	cStr :=
		`{
			"timestamp": 1559021720000000000,
			"proposer_num": 1,
			"period": 3000,
			"alternate_interval": 3000,
			"term_interval": 6000,
			"block_num": 20,
			"vote_unit_price": 1,
			"init_proposer": {
				"1": ["dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"]
			},
			"init_proposer_neturl": {
				"1": ["/ip4/127.0.0.1/tcp/47101/p2p/QmVxeNubpg1ZQjQT8W5yZC9fD7ZB1ViArwvyGUB53sqf8e"]
			},
			"bft_config":{}
		}`
	xconfig := &tdposConfig{}
	err := json.Unmarshal([]byte(cStr), xconfig)
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	t.Log("Config unmarshal", "v", xconfig)
}
