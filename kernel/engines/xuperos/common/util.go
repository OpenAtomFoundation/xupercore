package common

import (
	"encoding/json"
	"github.com/xuperchain/xupercore/lib/crypto/client"
)

// GetKVEngineType get kv engine type from xuper.json
func GetKVEngineType(config []byte) (string, error) {
	rootJSON := map[string]interface{}{}
	err := json.Unmarshal(config, &rootJSON)
	if err != nil {
		return "", err
	}
	kvEngineType := rootJSON["kvengine"]
	if kvEngineType == nil {
		return "default", nil
	}
	return kvEngineType.(string), nil
}

// GetCryptoType get crypto type from xuper.json
func GetCryptoType(config []byte) (string, error) {
	rootJSON := map[string]interface{}{}
	err := json.Unmarshal(config, &rootJSON)
	if err != nil {
		return "", err
	}
	cryptoType := rootJSON["crypto"]
	if cryptoType == nil {
		return client.CryptoTypeDefault, nil
	}
	return cryptoType.(string), nil
}
