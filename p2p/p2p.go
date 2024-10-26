package p2p

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/SyedMa3/peerlink/protocol"
	"github.com/SyedMa3/peerlink/utils"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	HandshakeProtocol     = "/handshake/1.0.0"
	MetadataProtocol      = "/metadata/1.0.0"
	FileTransferProtocol  = "/file-transfer/1.0.0"
	CompleteCheckProtocol = "/complete-check/1.0.0"
)

func HandleSend(ctx context.Context, node *Node, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("handleSend: file does not exist: %s", filePath)
	}

	err := node.generateWordsAndCid()
	if err != nil {
		return fmt.Errorf("handleSend: failed to generate words and CID: %w", err)
	}

	handshakeDone := make(chan bool)
	node.Host.SetStreamHandler(HandshakeProtocol, func(stream network.Stream) {
		go func() {
			handshakeDone <- true
			node.sharedKey, err = protocol.HandleHandshake(stream, node.words)
			if err != nil {
				fmt.Printf("handshake failed\n")
				panic(err)
			}
			fmt.Println("Handshake completed successfully")
			handshakeDone <- true
		}()
	})

	publishCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	err = node.PublishAddress(publishCtx)
	if err != nil {
		return fmt.Errorf("handleSend: failed to publish address to DHT: %w", err)
	}
	fmt.Println("Share the following four words with the receiver securely:")
	fmt.Println(strings.Join(node.words, "-"))
	<-handshakeDone

	metadataDone := make(chan bool)
	node.Host.SetStreamHandler(MetadataProtocol, func(stream network.Stream) {
		go func() {
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				fmt.Printf("handleSend: failed to get file info: %v", err)
				return
			}
			metadata := protocol.Metadata{
				Filename: filePath,
				Size:     fileInfo.Size(),
			}
			<-handshakeDone
			metadataDone <- true
			willReceive, err := protocol.SendMetadata(stream, metadata, node.sharedKey)
			if err != nil {
				fmt.Printf("handleSend: metadata exchange failed\n")
				panic(err)
			}
			if willReceive {
				metadataDone <- true
				return
			}
			metadataDone <- false
		}()
	})

	fmt.Println("\nWaiting for the receiver to connect and request the file...")
	<-metadataDone
	node.Host.SetStreamHandler(FileTransferProtocol, func(stream network.Stream) {
		go protocol.SendFile(stream, filePath, node.sharedKey)
	})
	willReceive := <-metadataDone
	if !willReceive {
		fmt.Println("Receiver declined the file transfer")
		return nil
	}

	completeCheckDone := make(chan bool)
	node.Host.SetStreamHandler(CompleteCheckProtocol, func(stream network.Stream) {
		go func() {
			protocol.ReceiveCompleteCheck(stream, node.sharedKey)
			completeCheckDone <- true
		}()
	})
	<-completeCheckDone

	return nil
}

func HandleReceive(ctx context.Context, node *Node, passphrase string) error {
	words := strings.Split(passphrase, "-")
	if len(words) != 5 {
		return fmt.Errorf("handleReceive: exactly five words are required")
	}

	err := node.setWordsAndCid(words)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to set words and CID: %w", err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	connectedSender, err := node.QueryAndConnect(queryCtx)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to query and connect to sender: %w", err)
	}

	err = startHandshake(ctx, node, connectedSender)
	if err != nil {
		return fmt.Errorf("handleReceive: handshake failed: %w", err)
	}
	fmt.Println("Handshake completed successfully")

	filename, err := startMetadataExchange(ctx, node, connectedSender)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to exchange metadata: %w", err)
	}

	receivedFilename, err := startFileReceive(ctx, node, connectedSender, filename)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to receive file: %w", err)
	}
	fmt.Printf("\nFile received successfully and saved as %s\n", receivedFilename)

	return nil
}

func startFileReceive(ctx context.Context, node *Node, senderID *peer.AddrInfo, filename string) (string, error) {
	file, err := utils.CheckFileExists(filename)
	if err != nil {
		return "", fmt.Errorf("startFileReceive: %w", err)
	}
	defer file.Close()

	stream, err := node.Host.NewStream(ctx, senderID.ID, FileTransferProtocol)
	if err != nil {
		return "", fmt.Errorf("startFileReceive: failed to create file transfer stream: %w", err)
	}

	err = protocol.ReceiveFile(stream, file, node.sharedKey)
	stream.Close()
	if err != nil {
		return "", fmt.Errorf("startFileReceive: failed to receive file: %w", err)
	}

	stream, err = node.Host.NewStream(ctx, senderID.ID, CompleteCheckProtocol)
	if err != nil {
		return "", fmt.Errorf("startFileReceive: failed to create complete check stream: %w", err)
	}
	protocol.SendCompleteCheck(stream, node.sharedKey)
	stream.Close()

	return file.Name(), nil
}

func startHandshake(ctx context.Context, node *Node, senderID *peer.AddrInfo) error {
	handshakeStream, err := node.Host.NewStream(ctx, senderID.ID, HandshakeProtocol)
	if err != nil {
		return fmt.Errorf("startHandshake: failed to create handshake stream: %w", err)
	}
	defer handshakeStream.Close()

	sessionKey, err := protocol.PerformHandshake(handshakeStream, node.words)
	if err != nil {
		return fmt.Errorf("startHandshake: handshake failed: %w", err)
	}

	node.sharedKey = sessionKey
	return nil
}

func startMetadataExchange(ctx context.Context, node *Node, senderID *peer.AddrInfo) (string, error) {
	metadataStream, err := node.Host.NewStream(ctx, senderID.ID, MetadataProtocol)
	if err != nil {
		return "", fmt.Errorf("startMetadataExchange: failed to create metadata stream: %w", err)
	}
	defer metadataStream.Close()

	// After handshake, receive metadata
	filename, accept, err := protocol.ReceiveMetadata(metadataStream, node.sharedKey)
	if err != nil {
		return "", fmt.Errorf("startMetadataExchange: failed to receive metadata: %w", err)
	}

	if accept {
		return filename, nil
	}

	return "", fmt.Errorf("startMetadataExchange: receiver declined the file transfer")
}
