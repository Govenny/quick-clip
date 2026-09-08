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
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

var (
	contentWriteMu   sync.Mutex
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procReplaceFileW = kernel32.NewProc("ReplaceFileW")
)

const replaceFileWriteThrough = 0x00000001

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

func ReadContent(path string, keys string) ([]any, error) {
	content, err := readContentFile(path, keys)
	if err == nil {
		return content, nil
	}

	backupPath := path + ".bak"
	backupContent, backupErr := readContentFile(backupPath, keys)
	if backupErr == nil {
		if restoreErr := restorePrimaryFromBackup(path, backupPath); restoreErr != nil {
			fmt.Printf("主数据读取失败，已加载备份但未能重建主文件: %v\n", restoreErr)
		} else {
			fmt.Printf("主数据读取失败，已从备份恢复: %s\n", backupPath)
		}
		return backupContent, nil
	}

	if errors.Is(err, os.ErrNotExist) && errors.Is(backupErr, os.ErrNotExist) {
		return []any{}, nil
	}

	return nil, fmt.Errorf("unable to load primary data (%w) or backup (%v)", err, backupErr)
}

func readContentFile(path string, keys string) ([]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decrypted, err := DecryptBytes(content, keys)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: %w", path, err)
	}

	var jsonCodec = jsoniter.ConfigCompatibleWithStandardLibrary
	var parsed []any
	if err := jsonCodec.Unmarshal(decrypted, &parsed); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return parsed, nil
}

// SaveContentBytes encrypts a JSON payload and atomically replaces the primary file.
// The previous valid primary file is retained as path + ".bak".
func SaveContentBytes(path string, keys string, byteData []byte) error {
	if !json.Valid(byteData) {
		return errors.New("content is not valid JSON")
	}

	resource, err := EncryptBytes(byteData, keys)
	if err != nil {
		return fmt.Errorf("encrypt content: %w", err)
	}

	contentWriteMu.Lock()
	defer contentWriteMu.Unlock()

	if err := writeEncryptedFile(path, resource, true); err != nil {
		return fmt.Errorf("atomic save content: %w", err)
	}
	return nil
}

func SaveContent(path string, keys string, content []any) error {
	byteData, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}
	return SaveContentBytes(path, keys, byteData)
}

func restorePrimaryFromBackup(path string, backupPath string) error {
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup: %w", err)
	}

	contentWriteMu.Lock()
	defer contentWriteMu.Unlock()

	// Keep the validated backup intact while replacing a missing or corrupt primary.
	return writeEncryptedFile(path, backup, false)
}

func writeEncryptedFile(path string, encrypted []byte, createBackup bool) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0600); err != nil {
		return fmt.Errorf("set temp file permissions: %w", err)
	}
	if _, err := tmp.Write(encrypted); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(tmpPath, path); err != nil {
			return fmt.Errorf("install initial data file: %w", err)
		}
		tmpPath = ""
		return nil
	} else if err != nil {
		return fmt.Errorf("stat primary data file: %w", err)
	}

	backupPath := ""
	if createBackup {
		backupPath = path + ".bak"
		if err := os.Remove(backupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove previous backup: %w", err)
		}
	}

	if err := replaceFile(path, tmpPath, backupPath); err != nil {
		return err
	}
	tmpPath = ""
	return nil
}

func replaceFile(primaryPath, replacementPath, backupPath string) error {
	primary, err := syscall.UTF16PtrFromString(primaryPath)
	if err != nil {
		return fmt.Errorf("encode primary path: %w", err)
	}
	replacement, err := syscall.UTF16PtrFromString(replacementPath)
	if err != nil {
		return fmt.Errorf("encode replacement path: %w", err)
	}

	var backup *uint16
	if backupPath != "" {
		backup, err = syscall.UTF16PtrFromString(backupPath)
		if err != nil {
			return fmt.Errorf("encode backup path: %w", err)
		}
	}

	result, _, callErr := procReplaceFileW.Call(
		uintptr(unsafe.Pointer(primary)),
		uintptr(unsafe.Pointer(replacement)),
		uintptr(unsafe.Pointer(backup)),
		replaceFileWriteThrough,
		0,
		0,
	)
	if result == 0 {
		return fmt.Errorf("ReplaceFileW: %w", callErr)
	}
	return nil
}
