package client

import (
	"fmt"
	"testing"
)

func Test_CreateCryptoClient(t *testing.T) {
	seed := []byte("Hello World")
	cc, err := CreateCryptoClient(CryptoTypeGM)
	if err != nil {
		t.Errorf("gen crypto client fail.err:%v", err)
	}
	key, err := cc.GenerateKeyBySeed(seed)
	if err != nil {
		t.Errorf("gen key fail.err:%v", err)
	}
	fmt.Println("key:", key)
}

func Test_CreateCryptoClientByPK(t *testing.T) {
	pubKey := "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571}"
	priKey := "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571,\"D\":29079635126530934056640915735344231956621504557963207107451663058887647996601}"
	cc, err := CreateCryptoClientFromJSONPrivateKey([]byte(pubKey))
	if err != nil {
		t.Errorf("create crypto client by pri key fail.err:%v", err)
	}
	msg := []byte("This is test msg")

	ecdPubKey, err := cc.GetEcdsaPublicKeyFromJsonStr(pubKey)
	if err != nil {
		t.Errorf("GetEcdsaPublicKeyFromJSON failed, err=%v\n", err)
		return
	}
	ecdPrivkey, err := cc.GetEcdsaPrivateKeyFromJsonStr(priKey)
	if err != nil {
		t.Errorf("GetEcdsaPrivateKeyFromJSON failed, err=%v\n", err)
		return
	}
	sign, err := cc.SignECDSA(ecdPrivkey, msg)
	if err != nil {
		t.Errorf("SignECDSA failed, err=%v\n", err)
		return
	}
	cc, err = CreateCryptoClientFromJSONPublicKey([]byte(pubKey))
	if err != nil {
		t.Errorf("create crypto client by pub key fail.err:%v", err)
	}
	ok, err := cc.VerifyECDSA(ecdPubKey, sign, msg)
	if err != nil {
		t.Errorf("VerifyECDSA data failed, err=%v\n", err)
		return
	}
	if !ok {
		t.Errorf("VerifyECDSA failed, result is not ok")
		return
	}
}
