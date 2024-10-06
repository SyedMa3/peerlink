package protocol

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/SyedMa3/peerlink/rw"
	"github.com/SyedMa3/peerlink/utils"
	"github.com/libp2p/go-libp2p/core/network"
)

func SendFile(stream network.Stream, filePath string, key []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	defer stream.Close()
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("sendFile: failed to open file: %v", err)
		return
	}
	defer file.Close()

	// Calculate the hash of the file
	hash, err := utils.CalculateFileHash(file)
	if err != nil {
		log.Printf("sendFile: failed to calculate file hash: %v", err)
		return
	}

	// Send the hash to the receiver
	hashWriter := bufio.NewWriter(stream)
	_, err = hashWriter.Write(hash)
	if err != nil {
		log.Printf("sendFile: failed to send file hash: %v", err)
		return
	}
	err = hashWriter.Flush()
	if err != nil {
		log.Printf("sendFile: failed to flush hash writer: %v", err)
		return
	}

	// Reset the file pointer to the beginning
	_, err = file.Seek(0, 0)
	if err != nil {
		log.Printf("sendFile: failed to reset file pointer: %v", err)
		return
	}

	fmt.Printf("Sending file: %s\n", filePath)

	w := bufio.NewWriter(stream)
	n, err := rw.WriteData(file, key, w)
	if err != nil {
		log.Printf("sendFile: failed to write data: %v", err)
		return
	}
	fmt.Println("sendFile: copied", n, "bytes to writer")
	fmt.Println("File sent successfully")
}

func ReceiveFile(stream network.Stream, file *os.File, key []byte) error {
	// Read the SHA256 checksum of the file from the stream
	checksum := make([]byte, 32) // SHA256 produces a 32-byte hash
	_, err := io.ReadFull(stream, checksum)
	if err != nil {
		log.Printf("receiveFile: failed to read file checksum: %v", err)
	}
	fmt.Printf("Received file checksum: %x\n", checksum)

	r := bufio.NewReader(stream)
	calculatedChecksum := rw.ReadData(r, key, file)

	fmt.Printf("Calculated checksum: %x\n", calculatedChecksum)
	if !bytes.Equal(checksum, calculatedChecksum) {
		return fmt.Errorf("receiveFile: received file checksum does not match")
	}

	return nil
}