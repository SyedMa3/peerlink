package p2p

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SyedMa3/peerlink/handshake"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
	"github.com/schollz/pake/v3"
)

const (
	HandshakeProtocol    = "/pake/handshake/1.0.0"
	FileTransferProtocol = "/file-transfer/1.0.0"
)

func NewHost(ctx context.Context) (host.Host, *dht.IpfsDHT, error) {
	h, err := libp2p.New(
	// libp2p.EnableHolePunching(),
	// libp2p.EnableAutoNATv2(),
	// libp2p.EnableAutoRelayWithStaticRelays(dht.GetDefaultBootstrapPeerAddrInfos()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("NewHost: failed to create libp2p host: %w", err)
	}

	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		return nil, nil, fmt.Errorf("NewHost: failed to create DHT: %w", err)
	}

	bootstrapPeers := dht.GetDefaultBootstrapPeerAddrInfos()

	for _, peerInfo := range bootstrapPeers {
		if err := h.Connect(ctx, peerInfo); err != nil {
			log.Printf("NewHost: failed to connect to bootstrap node %s: %v", peerInfo.ID, err)
		} else {
			log.Printf("NewHost: connected to bootstrap node: %s", peerInfo.ID)
		}
	}

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return nil, nil, fmt.Errorf("NewHost: failed to bootstrap DHT: %w", err)
	}

	return h, kademliaDHT, nil
}

func GenerateCID(filePath string) (cid.Cid, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("GenerateCID: failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return cid.Cid{}, fmt.Errorf("GenerateCID: failed to get file info: %w", err)
	}

	metadata := fmt.Sprintf("%s|%d", fileInfo.Name(), fileInfo.Size())
	metadataBytes := []byte(metadata)
	hash, err := multihash.Sum(metadataBytes, multihash.SHA2_256, -1)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("GenerateCID: failed to create multihash: %w", err)
	}

	return cid.NewCidV1(cid.Raw, hash), nil
}

func HandleSend(ctx context.Context, node *Node) error {
	fmt.Println("Enter the path of the file to send:")
	filePath, err := readInput()
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
	handshakeDone := make(chan struct{})

	// Set the HandshakeProtocol handler
	node.Host.SetStreamHandler(HandshakeProtocol, func(stream network.Stream) {
		go handleHandshake(stream, node, handshakeDone)
	})

	// Display the four words to the sender for sharing
	fmt.Println("\nShare the following four words with the receiver securely:")
	fmt.Println(strings.Join(node.words, " "))

	// Publish the CID to the DHT
	publishCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = node.PublishAddress(publishCtx)
	if err != nil {
		return fmt.Errorf("handleSend: failed to publish address to DHT: %w", err)
	}

	fmt.Println("handleSend: published address to DHT")

	// Wait for handshake to complete
	// <-handshakeDone

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Set the FileTransferProtocol handler after handshake
	node.Host.SetStreamHandler(FileTransferProtocol, func(stream network.Stream) {
		go sendFile(stream, filePath, node.sharedKey, wg)
	})

	fmt.Println("\nWaiting for the receiver to connect and request the file...")

	wg.Wait()

	return nil
}

func sendFile(stream network.Stream, filePath string, key []byte, wg *sync.WaitGroup) {
	defer stream.Close()
	defer wg.Done()

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("sendFile: failed to open file: %v", err)
		return
	}
	defer file.Close()

	fmt.Printf("Sending file: %s\n", filePath)

	// Create a buffer to read the file
	buffer := make([]byte, 4096)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			log.Printf("sendFile: failed to read file: %v", err)
			return
		}
		if n == 0 {
			break
		}

		// Encrypt the chunk using the shared key
		encryptedChunk, err := handshake.Encrypt(key, buffer[:n])
		if err != nil {
			log.Printf("sendFile: failed to encrypt chunk: %v", err)
			return
		}

		// Send the encrypted chunk
		_, err = stream.Write(encryptedChunk)
		if err != nil {
			log.Printf("sendFile: failed to send chunk: %v", err)
			return
		}
	}

	fmt.Println("File sent successfully")
}

func HandleReceive(ctx context.Context, node *Node) error {
	// Prompt for the four words
	fmt.Println("Enter the four words shared by the sender (separated by spaces):")
	wordsStr, err := readInput()
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

	// Query the DHT with a timeout
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	providers, err := node.QueryAddress(queryCtx)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to query DHT: %w", err)
	}

	if len(providers) == 0 {
		return fmt.Errorf("handleReceive: no providers found for the given CID")
	}

	fmt.Printf("Retrieved %d provider(s)\n", len(providers))

	var connectedSender peer.AddrInfo
	var connected bool

	for _, senderInfo := range providers {
		if err := node.Host.Connect(ctx, senderInfo); err != nil {
			fmt.Printf("Failed to connect to sender %s: %v\n", senderInfo.ID, err)
			continue
		}
		fmt.Printf("Connected to: %s\n", senderInfo.ID)
		connectedSender = senderInfo
		connected = true
		break
	}

	if !connected {
		return fmt.Errorf("handleReceive: failed to connect to any sender")
	}

	// Initiate handshake
	handshakeStream, err := node.Host.NewStream(ctx, connectedSender.ID, HandshakeProtocol)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to create handshake stream: %w", err)
	}

	weakKey := []byte(strings.Join(node.words, " "))

	// Initialize PAKE for sender with the generated key
	p, err := pake.InitCurve(weakKey, 0, "siec")
	if err != nil {
		return fmt.Errorf("handleReceive: failed to initialize PAKE: %w", err)
	}

	// Send receiver's PAKE bytes
	receiverBytes := p.Bytes()
	_, err = handshakeStream.Write(receiverBytes)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to send PAKE bytes: %w", err)
	}

	// Read sender's PAKE bytes
	senderBytes, err := readBytes(handshakeStream)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to read PAKE bytes: %w", err)
	}
	fmt.Println("senderBytes:", senderBytes)

	// Update PAKE with sender's bytes
	if err := p.Update(senderBytes); err != nil {
		return fmt.Errorf("handleReceive: failed to update PAKE: %w", err)
	}

	// Derive session key
	sessionKey, err := p.SessionKey()
	if err != nil {
		return fmt.Errorf("handleReceive: failed to derive session key: %w", err)
	}
	handshakeStream.Close()

	node.sharedKey = sessionKey

	fmt.Println("Handshake completed successfully")

	// Proceed with file transfer
	stream, err := node.Host.NewStream(ctx, connectedSender.ID, FileTransferProtocol)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to create file transfer stream: %w", err)
	}
	defer stream.Close()

	// Receive the file using the shared key
	if err := receiveFile(stream, node.sharedKey); err != nil {
		return fmt.Errorf("handleReceive: failed to receive file: %w", err)
	}

	fmt.Println("\nFile received and decrypted successfully")
	return nil
}

func receiveFile(stream network.Stream, key []byte) error {
	defer stream.Close()

	// Buffer to read encrypted data
	encryptedBuffer := make([]byte, 4096)

	for {
		n, err := stream.Read(encryptedBuffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("receiveFile: failed to read from stream: %w", err)
		}

		// Decrypt the chunk using the shared key
		decryptedChunk, err := handshake.Decrypt(key, encryptedBuffer[:n])
		if err != nil {
			return fmt.Errorf("receiveFile: failed to decrypt chunk: %w", err)
		}

		// Handle the decrypted data (e.g., save to file or output)
		fmt.Print(string(decryptedChunk))
	}

	return nil
}

func readInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("readInput: failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
}

func handleHandshake(stream network.Stream, node *Node, handshakeDone chan struct{}) {
	defer stream.Close()

	weakKey := []byte(strings.Join(node.words, " "))

	// Initialize PAKE for receiver
	p, err := pake.InitCurve(weakKey, 1, "siec")
	if err != nil {
		log.Printf("handleHandshake: failed to initialize PAKE: %v", err)
		return
	}

	// Read sender's PAKE bytes
	senderBytes, err := readBytes(stream)
	if err != nil {
		log.Printf("handleHandshake: failed to read sender bytes: %v", err)
		return
	}

	// Update PAKE with sender's bytes
	if err := p.Update(senderBytes); err != nil {
		log.Printf("handleHandshake: failed to update PAKE: %v", err)
		return
	}

	// Send receiver's PAKE bytes
	receiverBytes := p.Bytes()
	fmt.Println("receiverBytes:", receiverBytes)
	if _, err := stream.Write(receiverBytes); err != nil {
		log.Printf("handleHandshake: failed to send receiver bytes: %v", err)
		return
	}

	// Derive session key
	sessionKey, err := p.SessionKey()
	if err != nil {
		log.Printf("handleHandshake: failed to derive session key: %v", err)
		return
	}

	d, err := readBytes(stream)
	if err != io.EOF {
		if err != nil {
			log.Printf("handleHandshake: unexpected data received: %v", d)
			return
		}
		log.Printf("handleHandshake: failed to read sender bytes: %v", err)
		return
	}

	node.sharedKey = sessionKey
	log.Println("handleHandshake: PAKE authentication successful")
	close(handshakeDone)
}

func readBytes(stream network.Stream) ([]byte, error) {
	buf := make([]byte, 1024) // Adjust buffer size as needed
	n, err := stream.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}
