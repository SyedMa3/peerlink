package handshake

import (
	"fmt"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	bip39 "github.com/tyler-smith/go-bip39"
)

// GenerateRandomWords generates four random English words using the BIP39 English wordlist.
func GenerateRandomWords() ([]string, error) {
	// Generate 128 bits of entropy for generating a 12-word mnemonic
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return nil, fmt.Errorf("GenerateRandomWords: failed to generate entropy: %w", err)
	}

	// Generate mnemonic from entropy
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, fmt.Errorf("GenerateRandomWords: failed to generate mnemonic: %w", err)
	}

	// Split mnemonic into words and select the first four
	words := strings.Split(mnemonic, " ")
	if len(words) < 4 {
		return nil, fmt.Errorf("GenerateRandomWords: insufficient number of words generated")
	}

	return words[:4], nil
}

// GenerateCIDFromWordAndTime appends the first word and the current time (rounded up to 1 hour) to generate a CID.
func GenerateCIDFromWordAndTime(word string) (cid.Cid, error) {
	// Get the current time in UTC
	currentTime := time.Now().UTC()

	// Round up to the nearest hour
	roundedTime := currentTime.Truncate(time.Hour).Format(time.RFC3339)

	// Combine the first word with the rounded time
	data := fmt.Sprintf("%s|%s", word, roundedTime)
	dataBytes := []byte(data)

	// Create a multihash using SHA2-256
	hash, err := multihash.Sum(dataBytes, multihash.SHA2_256, -1)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("GenerateCIDFromWordAndTime: failed to create multihash: %w", err)
	}

	// Create a CID using the raw codec
	return cid.NewCidV1(cid.Raw, hash), nil
}