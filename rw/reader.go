package rw

import (
	"fmt"
	"io"

	"github.com/SyedMa3/peerlink/handshake"
)

type PReader struct {
	io.Reader
	key []byte
}

func NewPReader(r io.Reader, key []byte) *PReader {
	return &PReader{Reader: r, key: key}
}

func (r *PReader) Read(p []byte) (n int, err error) {
	// Read the length of the encrypted data
	lengthBytes := make([]byte, 4)
	n, err = io.ReadFull(r.Reader, lengthBytes)
	if err != nil {
		if err == io.EOF {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("failed to read data length: %w", err)
	}
	if n == 0 {
		return 0, io.EOF
	}

	// Convert the length bytes to uint32
	dataLength := uint32(lengthBytes[0])<<24 | uint32(lengthBytes[1])<<16 | uint32(lengthBytes[2])<<8 | uint32(lengthBytes[3])

	// Read the encrypted data
	encryptedData := make([]byte, dataLength)
	_, err = io.ReadFull(r.Reader, encryptedData)
	if err != nil {
		return 0, fmt.Errorf("failed to read encrypted data: %w", err)
	}

	// Decrypt the data
	decryptedData, err := handshake.Decrypt(r.key, encryptedData)
	if err != nil {
		return 0, fmt.Errorf("failed to decrypt data: %w", err)
	}

	// Copy the decrypted data to the output buffer
	n = copy(p, decryptedData)
	return n, nil
}