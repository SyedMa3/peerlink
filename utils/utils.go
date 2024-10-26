package utils

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/libp2p/go-libp2p/core/network"
)

func ReadInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("readInput: failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
}

func ReadBytes(stream network.Stream) ([]byte, error) {
	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// Encrypt encrypts the given data using the derived key
func Encrypt(key, data []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("Encrypt: invalid key length: expected 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Encrypt: failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("Encrypt: failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("Encrypt: failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt decrypts the given data using the derived key
func Decrypt(key, data []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("Decrypt: invalid key length: expected 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Decrypt: failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("Decrypt: failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("Decrypt: ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("Decrypt: failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// BytesToHex converts a byte slice to a hexadecimal string
func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

// HexToBytes converts a hexadecimal string to a byte slice
func HexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

func CalculateFileHash(file *os.File) ([]byte, error) {
	hash := sha256.New()
	_, err := io.Copy(hash, file)
	if err != nil {
		return nil, fmt.Errorf("calculateFileHash: failed to calculate file hash: %w", err)
	}
	return hash.Sum(nil), nil
}

func CheckFileExists(filename string) (*os.File, error) {
	// Check if file already exists
	ext := filepath.Ext(filename)
	baseFilename := strings.TrimSuffix(filename, ext)

	counter := 1
	for {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			break
		}
		filename = fmt.Sprintf("%s(%d)%s", baseFilename, counter, ext)
		counter++
	}

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("CheckFileExists: failed to create file %s: %v", filename, err)
	}
	return file, nil
}
