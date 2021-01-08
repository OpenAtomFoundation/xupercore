package xpoa

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalConfig(t *testing.T) {
	cStr := "{\"period\": 3000,\"block_num\": 10,\"init_proposer\": [{\"address\": \"f3prTg9itaZY6m48wXXikXdcxiByW7zgk\",\"neturl\": \"127.0.0.1:47102\"},{\"address\": \"U9sKwFmgJVfzgWcfAG47dKn1kLQTqeZN3\",\"neturl\": \"127.0.0.1:47103\"},{\"address\": \"RUEMFGDEnLBpnYYggnXukpVfR9Skm59ph\",\"neturl\": \"127.0.0.1:47104\"}]}"
	config := &xpoaConfig{}
	err := json.Unmarshal([]byte(cStr), config)
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	if config.Period != 3000 {
		t.Error("Config unmarshal err", "v", config.Period)
	}
}
