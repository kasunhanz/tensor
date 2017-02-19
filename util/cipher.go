package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	log "github.com/Sirupsen/logrus"
)

//Example usage:
//
//originalText := "encrypt this golang"
//fmt.Println(originalText)
//
// encrypt value to base64
//cryptoText := Encrypt(originalText)
//fmt.Println(cryptoText)
//
// encrypt base64 crypto to original value
//text := Decrypt(cryptoText)
//fmt.Printf(text)

// CipherEncrypt encrypts string to base64 crypto using AES
func CipherEncrypt(text string) string {

	//Return empty string if input text is empty
	if text == "" {
		return ""
	}

	// key := []byte(keyText)
	plaintext := []byte(text)

	block, err := aes.NewCipher([]byte(Config.Salt))
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		log.Errorln("Error occurred when reading AES blocks", err.Error())
		return ""
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext)
}

// CipherDecrypt from base64 to decrypted string
func CipherDecrypt(cryptoText string) string {
	//Return empty string if input cryptoText is also empty
	if cryptoText == "" {
		return ""
	}

	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher([]byte(Config.Salt))
	if err != nil {
		log.Errorln("Error occurred when generating new cipher block", err.Error())
		return ""
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		log.Errorln("Cipher text is too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext)
}
