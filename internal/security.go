package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	jsoniter "github.com/json-iterator/go"
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

func ReadContent(path string, keys string) []any {
	content, err := os.ReadFile(path)
	// 情况 A: 文件不存在，初始化并返回空数据
	if os.IsNotExist(err) {
		initialData := []byte("[]")
		// 将加密后的初始数据保存到磁盘，方便下次读取
		SaveContentBytes(path, keys, initialData)
		return []any{}
	} else if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		return nil
	}

	// 情况 B: 文件存在，解密
	decrypted, err := DecryptBytes(content, keys)
	if err != nil {
		fmt.Printf("解密失败 (可能是密钥不匹配或文件损坏): %v\n", err)
		return nil
	}

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	var tContent []any
	err = json.Unmarshal(decrypted, &tContent)
	if err != nil {
		fmt.Printf("JSON解析失败: %v\n", err)
		return nil
	}
	return tContent
}

func ReadContenttoBytes(path string, keys string) []byte {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// 文件不存在，创建文件并写入初始数据
		initialData := []byte("[]") // 空的JSON数组
		resource, err := EncryptBytes(initialData, keys)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		err = os.WriteFile(path, resource, 0644)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		content = initialData // 使用刚创建的初始数据
	} else if err != nil {
		// 其他读取错误
		fmt.Println(err)
		return nil
	}

	decrypted, err := DecryptBytes(content, keys)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return decrypted
}

// 辅助函数：直接保存字节流
func SaveContentBytes(path string, keys string, byteData []byte) {
	resource, err := EncryptBytes(byteData, keys)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, resource, 0644)
}

// 你的 SaveContent 也可以简化调用这个辅助函数
func SaveContent(path string, keys string, content []any) {
	byteData, err := json.Marshal(content)
	if err != nil || byteData == nil {
		return
	}
	SaveContentBytes(path, keys, byteData)
}
