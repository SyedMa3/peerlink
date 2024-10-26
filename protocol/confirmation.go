package protocol

import (
	"bufio"
	"fmt"

	"github.com/SyedMa3/peerlink/rw"
	"github.com/libp2p/go-libp2p/core/network"
)

func SendCompleteCheck(stream network.Stream, key []byte) error {
	writer := bufio.NewWriter(stream)
	pwriter := rw.NewPWriter(writer, key)
	_, err := pwriter.Write([]byte("y"))
	if err != nil {
		return fmt.Errorf("SendCompleteCheck: failed to send confirmation: %w", err)
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("SendCompleteCheck: failed to flush writer: %w", err)
	}
	return nil
}

func ReceiveCompleteCheck(stream network.Stream, key []byte) error {
	reader := rw.NewPReader(stream, key)
	confirmation := make([]byte, 1)
	_, err := reader.Read(confirmation)
	if err != nil {
		return fmt.Errorf("ReceiveCompleteCheck: failed to read confirmation: %w", err)
	}
	return nil
}
