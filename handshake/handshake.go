package handshake

import (
	"encoding/hex"
)

const (
	senderRole   = 0
	receiverRole = 1
	curveType    = "siec"
	keyLength    = 32 // 256 bits
)

// DeriveKey is now split into sender and receiver roles during the handshake
// No changes needed here as PAKE exchange is handled in p2p/p2p.go

// Encrypt encrypts the given data using the derived key
func Encrypt(key, data []byte) ([]byte, error) {
	// For simplicity, we'll use XOR encryption.
	// In a real-world scenario, you should use a proper encryption algorithm.
	encrypted := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		encrypted[i] = data[i] ^ key[i%len(key)]
	}
	return encrypted, nil
}

// Decrypt decrypts the given data using the derived key
func Decrypt(key, data []byte) ([]byte, error) {
	// XOR decryption (same as encryption)
	return Encrypt(key, data)
}

// BytesToHex converts a byte slice to a hexadecimal string
func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

// HexToBytes converts a hexadecimal string to a byte slice
func HexToBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}
