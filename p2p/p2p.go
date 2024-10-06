package p2p

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SyedMa3/peerlink/protocol"
	"github.com/SyedMa3/peerlink/utils"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	HandshakeProtocol    = "/handshake/1.0.0"
	MetadataProtocol     = "/metadata/1.0.0"
	FileTransferProtocol = "/file-transfer/1.0.0"
)

func HandleSend(ctx context.Context, node *Node) error {
	fmt.Println("Enter the path of the file to send:")
	filePath, err := utils.ReadInput()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("handleSend: file does not exist: %s", filePath)
	}

	err = node.generateWordsAndCid()
	if err != nil {
		return fmt.Errorf("handleSend: failed to generate words and CID: %w", err)
	}

	node.Host.SetStreamHandler(HandshakeProtocol, func(stream network.Stream) {
		go func() {
			node.sharedKey, err = protocol.HandleHandshake(stream, node.words)
			if err != nil {
				log.Printf("handleSend: handshake failed\n")
				panic(err)
			}
			fmt.Println("handleSend: handshake completed successfully")
		}()
	})

	fmt.Println("\nShare the following four words with the receiver securely:")
	fmt.Println(strings.Join(node.words, " "))

	publishCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	err = node.PublishAddress(publishCtx)
	if err != nil {
		return fmt.Errorf("handleSend: failed to publish address to DHT: %w", err)
	}
	fmt.Println("handleSend: published address to DHT")

	wg := &sync.WaitGroup{}
	wg.Add(1)

	node.Host.SetStreamHandler(FileTransferProtocol, func(stream network.Stream) {
		go protocol.SendFile(stream, filePath, node.sharedKey, wg)
	})
	fmt.Println("\nWaiting for the receiver to connect and request the file...")

	wg.Wait()

	return nil
}

func HandleReceive(ctx context.Context, node *Node) error {
	fmt.Println("Enter the four words shared by the sender (separated by spaces):")
	wordsStr, err := utils.ReadInput()
	if err != nil {
		log.Fatal(err)
	}
	words := strings.Split(wordsStr, " ")
	if len(words) != 4 {
		return fmt.Errorf("handleReceive: exactly four words are required")
	}

	err = node.setWordsAndCid(words)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to set words and CID: %w", err)
	}
	fmt.Printf("Generated CID: %s\n", node.cid.String())

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

	if err := startFileReceive(ctx, node, connectedSender); err != nil {
		return fmt.Errorf("handleReceive: failed to receive file: %w", err)
	}
	fmt.Println("\nFile received and decrypted successfully")

	return nil
}

func startFileReceive(ctx context.Context, node *Node, senderID *peer.AddrInfo) error {

	fmt.Print("Enter the filename to save the received file: ")
	filename, err := utils.ReadInput()
	if err != nil {
		return fmt.Errorf("startFileReceive: failed to read filename input: %w", err)
	}

	file, err := utils.CheckFileExists(filename)
	if err != nil {
		return fmt.Errorf("startFileReceive: %w", err)
	}
	defer file.Close()

	stream, err := node.Host.NewStream(ctx, senderID.ID, FileTransferProtocol)
	if err != nil {
		return fmt.Errorf("startFileReceive: failed to create file transfer stream: %w", err)
	}
	defer stream.Close()

	err = protocol.ReceiveFile(stream, file, node.sharedKey)
	if err != nil {
		return fmt.Errorf("startFileReceive: failed to receive file: %w", err)
	}

	stream.Close()

	return nil
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
