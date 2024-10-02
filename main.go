package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/SyedMa3/peerlink/p2p"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Create a new libp2p host with DHT
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, err := p2p.NewNode(ctx)
	if err != nil {
		log.Fatalf("main: failed to create node: %v", err)
	}
	defer node.Host.Close()

	// Prompt user for action
	fmt.Println("\nChoose an option:")
	fmt.Println("1. Send a file")
	fmt.Println("2. Receive a file")
	action, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	action = strings.TrimSpace(action)

	switch action {
	case "1":
		if err := p2p.HandleSend(ctx, node); err != nil {
			log.Fatalf("main: failed to handle send: %v", err)
		}
	case "2":
		if err := p2p.HandleReceive(ctx, node); err != nil {
			log.Fatalf("main: failed to handle receive: %v", err)
		}
	default:
		log.Fatal("main: invalid option. Please choose '1' or '2'.")
	}
}
