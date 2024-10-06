package rw

import (
	"fmt"
	"io"
	"os"

	"github.com/SyedMa3/peerlink/utils"
)

type PWriter struct {
	io.Writer
	key []byte
}

func NewPWriter(w io.Writer, key []byte) *PWriter {
	return &PWriter{Writer: w, key: key}
}

func (w *PWriter) Write(p []byte) (n int, err error) {
	// Encrypt the data before writing
	encryptedData, err := utils.Encrypt(w.key, p)
	if err != nil {
		return 0, fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Prepend the length of the encrypted data
	dataLength := uint32(len(encryptedData))
	lengthBytes := make([]byte, 4)
	lengthBytes[0] = byte(dataLength >> 24)
	lengthBytes[1] = byte(dataLength >> 16)
	lengthBytes[2] = byte(dataLength >> 8)
	lengthBytes[3] = byte(dataLength)

	// Write the length followed by the encrypted data
	_, err = w.Writer.Write(lengthBytes)
	if err != nil {
		return n, fmt.Errorf("failed to write data length: %w", err)
	}

	_, err = w.Writer.Write(encryptedData)
	if err != nil {
		return n, fmt.Errorf("failed to write encrypted data: %w", err)
	}

	return len(p), nil
}

func WriteData(file *os.File, key []byte, w io.Writer) (int64, error) {
	writer := NewPWriter(w, key)

	n, err := io.Copy(writer, file)
	if err != nil {
		return 0, err
	}
	return n, nil
}
