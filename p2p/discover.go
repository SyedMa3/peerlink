package p2p

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

func (n *Node) PublishAddress(ctx context.Context) error {
	log.Printf("PublishAddress: providing CID: %s", n.cid.String())
	if err := n.DHT.Provide(ctx, n.cid, true); err != nil {
		return fmt.Errorf("PublishAddress: failed to provide CID: %w", err)
	}
	return nil
}

func (n *Node) QueryAddress(ctx context.Context) ([]peer.AddrInfo, error) {
	log.Printf("QueryAddress: finding providers for CID: %s", n.cid.String())
	providers, err := n.DHT.FindProviders(ctx, n.cid)
	if err != nil {
		return nil, fmt.Errorf("QueryAddress: failed to find providers: %w", err)
	}
	return providers, nil
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
