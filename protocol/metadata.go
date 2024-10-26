package protocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/SyedMa3/peerlink/rw"
	"github.com/SyedMa3/peerlink/utils"
	"github.com/libp2p/go-libp2p/core/network"
)

type Metadata struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

func SendMetadata(stream network.Stream, metadata Metadata, key []byte) (bool, error) {
	defer stream.Close()

	writer := bufio.NewWriter(stream)
	reader := bufio.NewReader(stream)

	pwriter := rw.NewPWriter(writer, key)
	preader := rw.NewPReader(reader, key)

	// Serialize metadata to JSON
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return false, fmt.Errorf("SendMetadata: failed to marshal metadata: %w", err)
	}

	// Send metadata
	_, err = pwriter.Write(metadataBytes)
	if err != nil {
		return false, fmt.Errorf("SendMetadata: failed to write metadata: %w", err)
	}
	err = writer.Flush()
	if err != nil {
		return false, fmt.Errorf("SendMetadata: failed to flush writer: %w", err)
	}

	// Await confirmation from receiver
	confirmation := make([]byte, 1)
	_, err = preader.Read(confirmation)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("SendMetadata: failed to read confirmation: %w", err)
	}
	answer := strings.TrimSpace(strings.ToLower(string(confirmation)))

	if answer == "y" {
		return true, nil
	} else {
		return false, nil
	}
}

func ReceiveMetadata(stream network.Stream, key []byte) (bool, error) {
	defer stream.Close()

	// Initialize a buffered writer and reader
	writer := bufio.NewWriter(stream)
	reader := bufio.NewReader(stream)

	pwriter := rw.NewPWriter(writer, key)
	preader := rw.NewPReader(reader, key)

	// Read metadata JSON from the stream
	metadataBytes := make([]byte, 1024)
	_, err := preader.Read(metadataBytes)
	if err != nil {
		return false, fmt.Errorf("ReceiveMetadata: failed to read metadata: %w", err)
	}

	// Deserialize metadata
	var metadata Metadata
	err = json.Unmarshal(bytes.TrimRight(metadataBytes, "\x00"), &metadata)
	if err != nil {
		return false, fmt.Errorf("ReceiveMetadata: failed to unmarshal metadata: %w", err)
	}

	// Prompt user for confirmation (Assuming a synchronous prompt)
	fmt.Printf("Received file metadata:\nFilename: %s\nSize: %d bytes\nDo you want to receive this file? (y/n): ", metadata.Filename, metadata.Size)
	response, err := utils.ReadInput()
	if err != nil {
		return false, fmt.Errorf("ReceiveMetadata: failed to read user input: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))

	// Send confirmation back to sender
	if response != "y" {
		response = "n"
	}

	_, err = pwriter.Write([]byte(response))
	if err != nil {
		return false, fmt.Errorf("ReceiveMetadata: failed to send confirmation: %w", err)
	}
	err = writer.Flush()
	if err != nil {
		return false, fmt.Errorf("ReceiveMetadata: failed to flush writer: %w", err)
	}

	return response == "y", nil
}
