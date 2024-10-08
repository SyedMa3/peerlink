package p2p

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
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
