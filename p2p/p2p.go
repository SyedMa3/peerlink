package p2p

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SyedMa3/peerlink/rw"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/schollz/pake/v3"
)

const (
	HandshakeProtocol    = "/pake/handshake/1.0.0"
	FileTransferProtocol = "/file-transfer/1.0.0"
)

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

	node.Host.SetStreamHandler(HandshakeProtocol, func(stream network.Stream) {
		go handleHandshake(stream, node)
	})

	fmt.Println("\nShare the following four words with the receiver securely:")
	fmt.Println(strings.Join(node.words, " "))

	publishCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = node.PublishAddress(publishCtx)
	if err != nil {
		return fmt.Errorf("handleSend: failed to publish address to DHT: %w", err)
	}

	fmt.Println("handleSend: published address to DHT")

	wg := &sync.WaitGroup{}
	wg.Add(1)

	node.Host.SetStreamHandler(FileTransferProtocol, func(stream network.Stream) {
		go sendFile(stream, filePath, node.sharedKey, wg)
	})

	fmt.Println("\nWaiting for the receiver to connect and request the file...")

	wg.Wait()

	return nil
}

func sendFile(stream network.Stream, filePath string, key []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	defer stream.Close()
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("sendFile: failed to open file: %v", err)
		return
	}
	defer file.Close()

	// Calculate the hash of the file
	hash, err := calculateFileHash(file)
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

	rw := bufio.NewWriter(stream)
	writeData(file, key, rw)
}

func calculateFileHash(file *os.File) ([]byte, error) {
	hash := sha256.New()
	_, err := io.Copy(hash, file)
	if err != nil {
		return nil, fmt.Errorf("calculateFileHash: failed to calculate file hash: %w", err)
	}
	return hash.Sum(nil), nil
}

func writeData(file *os.File, key []byte, w *bufio.Writer) {
	writer := rw.NewPWriter(w, key)

	n, err := io.Copy(writer, file)
	if err != nil {
		log.Printf("writeData: failed to copy data to writer: %v", err)
	}
	fmt.Println("writeData: copied", n, "bytes to writer")

	fmt.Println("File sent successfully")
}

func HandleReceive(ctx context.Context, node *Node) error {
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

	handshakeStream, err := node.Host.NewStream(ctx, connectedSender.ID, HandshakeProtocol)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to create handshake stream: %w", err)
	}

	weakKey := []byte(strings.Join(node.words, " "))

	p, err := pake.InitCurve(weakKey, 0, "siec")
	if err != nil {
		return fmt.Errorf("handleReceive: failed to initialize PAKE: %w", err)
	}

	receiverBytes := p.Bytes()
	_, err = handshakeStream.Write(receiverBytes)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to send PAKE bytes: %w", err)
	}

	senderBytes, err := readBytes(handshakeStream)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to read PAKE bytes: %w", err)
	}

	if err := p.Update(senderBytes); err != nil {
		return fmt.Errorf("handleReceive: failed to update PAKE: %w", err)
	}

	sessionKey, err := p.SessionKey()
	if err != nil {
		return fmt.Errorf("handleReceive: failed to derive session key: %w", err)
	}
	handshakeStream.Close()

	node.sharedKey = sessionKey

	fmt.Println("Handshake completed successfully")

	stream, err := node.Host.NewStream(ctx, connectedSender.ID, FileTransferProtocol)
	if err != nil {
		return fmt.Errorf("handleReceive: failed to create file transfer stream: %w", err)
	}
	defer stream.Close()

	if err := receiveFile(stream, node.sharedKey); err != nil {
		return fmt.Errorf("handleReceive: failed to receive file: %w", err)
	}

	fmt.Println("\nFile received and decrypted successfully")
	return nil
}

func receiveFile(stream network.Stream, key []byte) error {
	fmt.Println("receiveFile: starting to read from stream")

	// Read the SHA256 checksum of the file from the stream
	checksum := make([]byte, 32) // SHA256 produces a 32-byte hash
	_, err := io.ReadFull(stream, checksum)
	if err != nil {
		return fmt.Errorf("receiveFile: failed to read file checksum: %w", err)
	}
	fmt.Printf("Received file checksum: %x\n", checksum)

	rw := bufio.NewReader(stream)
	calculatedChecksum := readData(rw, key)

	fmt.Printf("Calculated checksum: %x\n", calculatedChecksum)
	if !bytes.Equal(checksum, calculatedChecksum) {
		return fmt.Errorf("receiveFile: received file checksum does not match")
	}

	stream.Close()

	return nil
}

func readData(r *bufio.Reader, key []byte) []byte {
	reader := rw.NewPReader(r, key)

	file, err := os.Create("tmp")
	if err != nil {
		log.Printf("readData: failed to create tmp file: %v", err)
		return nil
	}
	defer file.Close()

	checksum := sha256.New()
	n, err := io.Copy(io.MultiWriter(file, checksum), reader)
	if err != nil {
		log.Printf("readData: failed to copy data to tmp file: %v", err)
		return nil
	}
	fmt.Println("readData: copied", n, "bytes to stdout")

	return checksum.Sum(nil)
}

func readInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("readInput: failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
}

func handleHandshake(stream network.Stream, node *Node) {
	defer stream.Close()

	weakKey := []byte(strings.Join(node.words, " "))

	p, err := pake.InitCurve(weakKey, 1, "siec")
	if err != nil {
		log.Printf("handleHandshake: failed to initialize PAKE: %v", err)
		return
	}

	senderBytes, err := readBytes(stream)
	if err != nil {
		log.Printf("handleHandshake: failed to read sender bytes: %v", err)
		return
	}

	if err := p.Update(senderBytes); err != nil {
		log.Printf("handleHandshake: failed to update PAKE: %v", err)
		return
	}

	receiverBytes := p.Bytes()
	fmt.Println("receiverBytes:", receiverBytes)
	if _, err := stream.Write(receiverBytes); err != nil {
		log.Printf("handleHandshake: failed to send receiver bytes: %v", err)
		return
	}

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
}

func readBytes(stream network.Stream) ([]byte, error) {
	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}
