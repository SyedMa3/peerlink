package p2p

import (
	"context"
	"fmt"

	"github.com/SyedMa3/peerlink/handshake"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
)

type Node struct {
	Host      host.Host
	DHT       *dht.IpfsDHT
	words     []string
	cid       cid.Cid
	sharedKey []byte // {{ edit_1 }} Added sharedKey to store the PAKE-derived key
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
