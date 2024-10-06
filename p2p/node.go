package p2p

import (
	"context"
	"fmt"
	"log"

	"github.com/SyedMa3/peerlink/protocol"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Node struct {
	Host      host.Host
	DHT       *dht.IpfsDHT
	words     []string
	cid       cid.Cid
	sharedKey []byte
}

func NewNode(ctx context.Context) (*Node, error) {
	h, kademliaDHT, err := NewHost(ctx)
	if err != nil {
		return nil, fmt.Errorf("NewNode: failed to create host: %w", err)
	}

	return &Node{
		Host: h,
		DHT:  kademliaDHT,
	}, nil
}

func NewHost(ctx context.Context) (host.Host, *dht.IpfsDHT, error) {
	h, err := libp2p.New(
		libp2p.EnableHolePunching(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableAutoRelayWithStaticRelays(dht.GetDefaultBootstrapPeerAddrInfos()),
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

func (n *Node) QueryAndConnect(ctx context.Context) (*peer.AddrInfo, error) {
	providers, err := n.QueryAddress(ctx)
	if err != nil {
		return nil, fmt.Errorf("QueryAndConnect: failed to query DHT: %w", err)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("QueryAndConnect: no providers found for the given CID")
	}

	fmt.Printf("Retrieved %d provider(s)\n", len(providers))

	var connectedSender peer.AddrInfo
	var connected bool

	for _, senderInfo := range providers {
		if err := n.Host.Connect(ctx, senderInfo); err != nil {
			fmt.Printf("Failed to connect to sender %s: %v\n", senderInfo.ID, err)
			continue
		}
		fmt.Printf("Connected to: %s\n", senderInfo.ID)
		connectedSender = senderInfo
		connected = true
		break
	}

	if !connected {
		return nil, fmt.Errorf("QueryAndConnect: failed to connect to any sender")
	}

	return &connectedSender, nil
}

func (n *Node) generateWordsAndCid() error {
	words, err := protocol.GenerateRandomWords()
	if err != nil {
		return fmt.Errorf("generateWordsAndCid: failed to generate random words: %w", err)
	}
	cid, err := protocol.GenerateCIDFromWordAndTime(words[0])
	if err != nil {
		return fmt.Errorf("generateWordsAndCid: failed to generate CID: %w", err)
	}
	n.words = words
	n.cid = cid
	return nil
}

func (n *Node) setWordsAndCid(words []string) error {
	cid, err := protocol.GenerateCIDFromWordAndTime(words[0])
	if err != nil {
		return fmt.Errorf("setWordsAndCid: failed to generate CID: %w", err)
	}
	n.cid = cid
	n.words = words
	return nil
}
