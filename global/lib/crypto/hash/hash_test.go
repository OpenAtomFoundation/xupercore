package hash

import (
	"testing"
)

func Test_Hash(t *testing.T) {
	msg := []byte("this is a test msg")

	sha256 := UsingSha256(msg)
	doubleSha256 := DoubleSha256(msg)
	ripemd160 := UsingRipemd160(msg)

	seed := []byte("this is seed")
	hmac512 := HashUsingHmac512(seed, msg)

	t.Logf("sha256=%v, doubleSha256=%v, ripemd160=%v, hmac512=%v\n", sha256, doubleSha256, ripemd160, hmac512)
}
