package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Encrypt encrypts plaintext using AES-256-CBC with a random IV.
// The IV is prepended to the ciphertext and the whole is base64 encoded.
// Key should be 32 bytes.
func Encrypt(plaintext, key string) (string, error) {
	if len(key) != 32 {
		return "", errors.New("key must be 32 bytes")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padtext := make([]byte, len(plaintext)+padding)
	copy(padtext, []byte(plaintext))
	for i := len(plaintext); i < len(padtext); i++ {
		padtext[i] = byte(padding)
	}

	ciphertext := make([]byte, len(iv)+len(padtext))
	copy(ciphertext[:aes.BlockSize], iv)

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], padtext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// EncryptBytes encrypts plaintext using AES-256-CBC with a random IV.
// The IV is prepended to the ciphertext.
// Key should be 32 bytes.
// Plaintext and output are []byte.
func EncryptBytes(plaintext []byte, key string) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padtext := make([]byte, len(plaintext)+padding)
	copy(padtext, plaintext)
	for i := len(plaintext); i < len(padtext); i++ {
		padtext[i] = byte(padding)
	}

	ciphertext := make([]byte, len(iv)+len(padtext))
	copy(ciphertext[:aes.BlockSize], iv)

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], padtext)

	return ciphertext, nil
}

// Decrypt decrypts the ciphertext using AES-256-CBC.
// Expects the IV prepended to the ciphertext.
// Ciphertext is provided as []byte (already decoded from base64).
func Decrypt(ciphertext, key string) (string, error) {
	if len(key) != 32 {
		return "", errors.New("key must be 32 bytes")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	if len(data) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	iv := data[:aes.BlockSize]
	ciphertextBytes := data[aes.BlockSize:]

	if len(ciphertextBytes)%aes.BlockSize != 0 {
		return "", errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertextBytes, ciphertextBytes)

	padding := int(ciphertextBytes[len(ciphertextBytes)-1])
	if padding > aes.BlockSize || padding == 0 {
		return "", errors.New("invalid padding")
	}

	for i := len(ciphertextBytes) - padding; i < len(ciphertextBytes); i++ {
		if ciphertextBytes[i] != byte(padding) {
			return "", errors.New("invalid padding")
		}
	}

	return string(ciphertextBytes[:len(ciphertextBytes)-padding]), nil
}

// DecryptBytes decrypts the ciphertext using AES-256-CBC.
// Expects the IV prepended to the ciphertext.
// Ciphertext is provided as []byte (already decoded from base64).
func DecryptBytes(ciphertext []byte, key string) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}

	data := ciphertext

	if len(data) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	iv := data[:aes.BlockSize]
	ciphertextBytes := data[aes.BlockSize:]

	if len(ciphertextBytes)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertextBytes, ciphertextBytes)

	padding := int(ciphertextBytes[len(ciphertextBytes)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, errors.New("invalid padding")
	}

	for i := len(ciphertextBytes) - padding; i < len(ciphertextBytes); i++ {
		if ciphertextBytes[i] != byte(padding) {
			return nil, errors.New("invalid padding")
		}
	}

	return ciphertextBytes[:len(ciphertextBytes)-padding], nil
}
