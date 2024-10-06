package protocol

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/SyedMa3/peerlink/utils"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/schollz/pake/v3"
)

func HandleHandshake(stream network.Stream, words []string) ([]byte, error) {
	defer stream.Close()

	weakKey := []byte(strings.Join(words, " "))

	p, err := pake.InitCurve(weakKey, 1, "siec")
	if err != nil {
		log.Printf("handleHandshake: failed to initialize PAKE: %v", err)
		return nil, err
	}

	senderBytes, err := utils.ReadBytes(stream)
	if err != nil {
		log.Printf("handleHandshake: failed to read sender bytes: %v", err)
		return nil, err
	}

	if err := p.Update(senderBytes); err != nil {
		log.Printf("handleHandshake: failed to update PAKE: %v", err)
		return nil, err
	}

	receiverBytes := p.Bytes()
	if _, err := stream.Write(receiverBytes); err != nil {
		log.Printf("handleHandshake: failed to send receiver bytes: %v", err)
		return nil, err
	}

	sessionKey, err := p.SessionKey()
	if err != nil {
		log.Printf("handleHandshake: failed to derive session key: %v", err)
		return nil, err
	}

	d, err := utils.ReadBytes(stream)
	if err != io.EOF {
		if err != nil {
			log.Printf("handleHandshake: unexpected data received: %v", d)
			return nil, err
		}
		log.Printf("handleHandshake: failed to read sender bytes: %v", err)
		return nil, err
	}

	return sessionKey, nil
}

func PerformHandshake(stream network.Stream, words []string) ([]byte, error) {
	weakKey := []byte(strings.Join(words, " "))

	p, err := pake.InitCurve(weakKey, 0, "siec")
	if err != nil {
		return nil, fmt.Errorf("performHandshake: failed to initialize PAKE: %w", err)
	}

	receiverBytes := p.Bytes()
	_, err = stream.Write(receiverBytes)
	if err != nil {
		return nil, fmt.Errorf("performHandshake: failed to send PAKE bytes: %w", err)
	}

	senderBytes, err := utils.ReadBytes(stream)
	if err != nil {
		return nil, fmt.Errorf("performHandshake: failed to read PAKE bytes: %w", err)
	}

	if err := p.Update(senderBytes); err != nil {
		return nil, fmt.Errorf("performHandshake: failed to update PAKE: %w", err)
	}

	sessionKey, err := p.SessionKey()
	if err != nil {
		return nil, fmt.Errorf("performHandshake: failed to derive session key: %w", err)
	}

	return sessionKey, nil
}
