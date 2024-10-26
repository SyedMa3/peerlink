package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/SyedMa3/peerlink/p2p"
	"github.com/briandowns/spinner"
	"github.com/urfave/cli/v2"
)

func main() {
	// Create a new libp2p host with DHT
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("Starting PeerLink...\n\n")

	app := &cli.App{
		Name:  "peerlink",
		Usage: "A peer-to-peer file sharing application",
		Commands: []*cli.Command{
			{
				Name:      "send",
				Usage:     "Send a file",
				ArgsUsage: "<filename>",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						return fmt.Errorf("filename is required")
					}
					filename := c.Args().First()
					node, err := initNode(ctx)
					if err != nil {
						return fmt.Errorf("failed to initialize node: %v", err)
					}
					defer node.Host.Close()
					return p2p.HandleSend(ctx, node, filename)
				},
			},
			{
				Name:      "receive",
				Usage:     "Receive a file",
				ArgsUsage: "<input-passphrase>",
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						return fmt.Errorf("input passphrase is required")
					}
					passphrase := c.Args().First()
					node, err := initNode(ctx)
					if err != nil {
						return fmt.Errorf("failed to initialize node: %v", err)
					}
					defer node.Host.Close()
					return p2p.HandleReceive(ctx, node, passphrase)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func initNode(ctx context.Context) (*p2p.Node, error) {
	s := spinner.New(spinner.CharSets[7], 100*time.Millisecond)
	s.Suffix = " Connecting to bootstrap nodes...\n"
	s.FinalMSG = "Connected to bootstrap nodes!\n"
	s.Start()
	defer s.Stop()
	return p2p.NewNode(ctx)
}
