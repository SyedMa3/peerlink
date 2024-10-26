package p2p

import (
	"context"
	"fmt"

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
		return nil, fmt.Errorf("failed to create host: %w", err)
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
		return nil, nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	bootstrapPeers := dht.GetDefaultBootstrapPeerAddrInfos()
	count := 0

	for _, peerInfo := range bootstrapPeers {
		if err := h.Connect(ctx, peerInfo); err != nil {
			fmt.Printf("failed to connect to bootstrap node %s: %v", peerInfo.ID, err)
		} else {
			count++
		}
	}

	if count == 0 {
		return nil, nil, fmt.Errorf("failed to connect to any bootstrap nodes")
	}
	fmt.Printf("Connected to %d bootstrap nodes\n", count)

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	return h, kademliaDHT, nil
}

func (n *Node) QueryAndConnect(ctx context.Context) (*peer.AddrInfo, error) {
	providers, err := n.QueryAddress(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query DHT: %w", err)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers found for the given CID")
	}

	fmt.Printf("Retrieved %d provider(s)\n", len(providers))

	var connectedSender peer.AddrInfo
	var connected bool

	for _, senderInfo := range providers {
		if err := n.Host.Connect(ctx, senderInfo); err != nil {
			fmt.Printf("failed to connect to sender %s: %v\n", senderInfo.ID, err)
			continue
		}
		connectedSender = senderInfo
		connected = true
		break
	}

	if !connected {
		return nil, fmt.Errorf("failed to connect to any sender")
	}

	return &connectedSender, nil
}

func (n *Node) generateWordsAndCid() error {
	words, err := protocol.GenerateRandomWords()
	if err != nil {
		return fmt.Errorf("failed to generate random words: %w", err)
	}
	cid, err := protocol.GenerateCIDFromWordAndTime(words[0])
	if err != nil {
		return fmt.Errorf("failed to generate CID: %w", err)
	}
	n.words = words
	n.cid = cid
	return nil
}

func (n *Node) setWordsAndCid(words []string) error {
	cid, err := protocol.GenerateCIDFromWordAndTime(words[0])
	if err != nil {
		return fmt.Errorf("failed to generate CID: %w", err)
	}
	n.cid = cid
	n.words = words
	return nil
}
