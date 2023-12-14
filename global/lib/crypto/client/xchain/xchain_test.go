package eccdefault

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func readKey(path string) ([]byte, []byte, []byte, error) {
	addr, err := ioutil.ReadFile(path + "/address")
	if err != nil {
		fmt.Printf("GetAccInfoFromFile error load address error = %v", err)
		return nil, nil, nil, err
	}
	pubkey, err := ioutil.ReadFile(path + "/public.key")
	if err != nil {
		fmt.Printf("GetAccInfoFromFile error load pubkey error = %v", err)
		return nil, nil, nil, err
	}
	prikey, err := ioutil.ReadFile(path + "/private.key")
	if err != nil {
		fmt.Printf("GetAccInfoFromFile error load prikey error = %v", err)
		return nil, nil, nil, err
	}
	return addr, pubkey, prikey, err
}

func cleanKey(path string) {
	os.Remove(path + "/address")
	os.Remove(path + "/public.key")
	os.Remove(path + "/private.key")
}

func Test_EccDefault(t *testing.T) {
	xcc := GetInstance()

	err := xcc.ExportNewAccount("./")
	if err != nil {
		t.Error("generate key failed")
		return
	}
	addr, pub, priv, err := readKey("./")
	if err != nil {
		t.Error("read key failed")
		return
	}
	t.Logf("created key, address=%s, pub=%s\n", addr, pub)
	defer cleanKey("./")

	msg := []byte("this is a test msg")

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
