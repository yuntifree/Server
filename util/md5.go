package util

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"github.com/satori/go.uuid"
)

//GetMD5Hash return hex md5 of text
func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

//GenUUID gen uuid
func GenUUID() string {
	u := uuid.NewV4()
	return u.String()
}

//GenSaltPasswd calc new password with salt
func GenSaltPasswd(password, salt string) string {
	return GetMD5Hash(password + salt)
}

//GenSalt gen 32 byte hex string
func GenSalt() string {
	uuid := GenUUID()
	return strings.Join(strings.Split(uuid, "-"), "")
}
