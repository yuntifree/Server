package util

import (
	"encoding/base64"
	"testing"
)

var key = []byte("1234567890123456")
var src = []byte("abcdefghigklmnopqrstuvwxyz0123456789")
var iv = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var encrypted = "8Z3dZzqn05FmiuBLowExK0CAbs4TY2GorC2dDPVlsn/tP+VuJGePqIMv1uSaVErr"

func Test_AesEncrypt(t *testing.T) {
	dst, err := AesEncrypt(src, key, iv)
	if err != nil || base64.StdEncoding.EncodeToString(dst) != encrypted {
		t.Error("AesEncrypt check failed!")
	}
}

func Test_AesDecrypt(t *testing.T) {
	ciphertext, _ := base64.StdEncoding.DecodeString(encrypted)
	dst, err := AesDecrypt(ciphertext, key, iv)
	if err != nil || string(dst) != string(src) {
		t.Error("AesDecrypt check failed!")
	}
}
