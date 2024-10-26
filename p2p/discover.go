package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (n *Node) PublishAddress(ctx context.Context) error {
	s := spinner.New(spinner.CharSets[7], 100*time.Millisecond)
	s.Suffix = " Publishing address to DHT...\n"
	s.Start()
	defer s.Stop()
	if err := n.DHT.Provide(ctx, n.cid, true); err != nil {
		return fmt.Errorf("PublishAddress: failed to provide CID: %w", err)
	}
	s.FinalMSG = "Published address to DHT!\n\n"
	return nil
}

func (n *Node) QueryAddress(ctx context.Context) ([]peer.AddrInfo, error) {
	s := spinner.New(spinner.CharSets[7], 100*time.Millisecond)
	s.Suffix = " Querying DHT for address...\n"
	s.Start()
	defer s.Stop()
	providers, err := n.DHT.FindProviders(ctx, n.cid)
	if err != nil {
		s.FinalMSG = "No providers found\n"
		return nil, fmt.Errorf("QueryAddress: failed to find providers: %w", err)
	}
	s.FinalMSG = fmt.Sprintf("Found %d providers!\n", len(providers))
	return providers, nil
}
