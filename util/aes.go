package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"log"
)

//AesDecrypt implement AES-128-CBC decrypt
func AesDecrypt(src, key, iv []byte) (dst []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("AesDecrypt NewCiper failed:%v", err)
		return
	}
	if len(dst)%aes.BlockSize != 0 {
		err = errors.New("input size illegal")
		return
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	dst = make([]byte, len(src))
	mode.CryptBlocks(dst, src)
	dst = pkcs7Unpadding(dst)
	return
}

//AesEncrypt implement AES-128-CBC encrypt
func AesEncrypt(src, key, iv []byte) (dst []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("AesEncrypt NewCipher failed:%v", err)
		return
	}
	src = pkcs7Padding(src, aes.BlockSize)
	mode := cipher.NewCBCEncrypter(block, iv)
	dst = make([]byte, len(src))
	mode.CryptBlocks(dst, src)
	return
}

func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pkcs7Unpadding(plaintext []byte) []byte {
	length := len(plaintext)
	unpadding := int(plaintext[length-1])
	return plaintext[:(length - unpadding)]
}
