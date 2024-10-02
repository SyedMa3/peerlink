package p2p

import (
	"context"
	"fmt"
	"log"

	"github.com/SyedMa3/peerlink/handshake"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
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

func (n *Node) generateWordsAndCid() error {
	words, err := handshake.GenerateRandomWords()
	if err != nil {
		return fmt.Errorf("generateWordsAndCid: failed to generate random words: %w", err)
	}
	cid, err := handshake.GenerateCIDFromWordAndTime(words[0])
	if err != nil {
		return fmt.Errorf("generateWordsAndCid: failed to generate CID: %w", err)
	}
	n.words = words
	n.cid = cid
	return nil
}

func (n *Node) setWordsAndCid(words []string) error {
	cid, err := handshake.GenerateCIDFromWordAndTime(words[0])
	if err != nil {
		return fmt.Errorf("setWordsAndCid: failed to generate CID: %w", err)
	}
	n.cid = cid
	n.words = words
	return nil
}
