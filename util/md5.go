package util

import (
	"crypto/md5"
	"encoding/hex"
)

//GetMD5Hash return hex md5 of text
func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
