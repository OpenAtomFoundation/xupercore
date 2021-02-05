package eccdefault

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

var (
	keypath = "./testkey"
)

func generateKey() error {
	err := os.Mkdir(keypath, os.ModePerm)
	if err != nil {
		return err
	}
	xcc := &XchainCryptoClient{}
	err = xcc.ExportNewAccount(keypath)
	return err
}

func readKey() ([]byte, []byte, []byte, error) {
	addr, err := ioutil.ReadFile(keypath + "/address")
	if err != nil {
		fmt.Printf("GetAccInfoFromFile error load address error = %v", err)
		return nil, nil, nil, err
	}
	pubkey, err := ioutil.ReadFile(keypath + "/public.key")
	if err != nil {
		fmt.Printf("GetAccInfoFromFile error load pubkey error = %v", err)
		return nil, nil, nil, err
	}
	prikey, err := ioutil.ReadFile(keypath + "/private.key")
	if err != nil {
		fmt.Printf("GetAccInfoFromFile error load prikey error = %v", err)
		return nil, nil, nil, err
	}
	return addr, pubkey, prikey, err
}

func cleanKey() {
	os.Remove(keypath + "/address")
	os.Remove(keypath + "/public.key")
	os.Remove(keypath + "/private.key")
	os.Remove(keypath)
}

func Test_EccDefault(t *testing.T) {
	err := generateKey()
	if err != nil {
		t.Error("generate key failed")
		return
	}
	addr, pub, priv, err := readKey()
	if err != nil {
		t.Error("read key failed")
		return
	}
	t.Logf("created key, address=%s, pub=%s\n", addr, pub)
	defer cleanKey()

	msg := []byte("this is a test msg")

	xcc := &XchainCryptoClient{}
	pubkey, err := xcc.GetEcdsaPublicKeyFromJsonStr(string(pub[:]))
	if err != nil {
		t.Errorf("GetEcdsaPublicKeyFromJSON failed, err=%v\n", err)
		return
	}
	privkey, err := xcc.GetEcdsaPrivateKeyFromJsonStr(string(priv[:]))
	if err != nil {
		t.Errorf("GetEcdsaPrivateKeyFromJSON failed, err=%v\n", err)
		return
	}

	// test encrypt and decrypt
	ciper, err := xcc.EncryptByEcdsaKey(pubkey, msg)
	if err != nil {
		t.Errorf("encrypt data failed, err=%v\n", err)
		return
	}

	decode, err := xcc.DecryptByEcdsaKey(privkey, ciper)
	if err != nil {
		t.Errorf("Decrypt data failed, err=%v\n", err)
		return
	}

	if bytes.Compare(msg, decode) != 0 {
		t.Errorf("Decrypt data is invalid, decoded=%s\n", string(decode))
		return
	}

	// test sign and verify
	sign, err := xcc.SignECDSA(privkey, msg)
	if err != nil {
		t.Errorf("SignECDSA failed, err=%v\n", err)
		return
	}

	ok, err := xcc.VerifyECDSA(pubkey, sign, msg)
	if err != nil {
		t.Errorf("VerifyECDSA data failed, err=%v\n", err)
		return
	}
	if !ok {
		t.Errorf("VerifyECDSA failed, result is not ok")
		return
	}
}
