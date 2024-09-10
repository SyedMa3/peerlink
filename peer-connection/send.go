package peerconnection

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

func SendFile() {
	var fileName string
	wg := sync.WaitGroup{}
	wg.Add(1)

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun1.l.google.com:19302",
				},
			},
		},
	}

	pconn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatal("Failed to create peer connection:", err)
	}
	defer func() {
		if err := pconn.Close(); err != nil {
			log.Println("Failed to close peer connection:", err)
		}
	}()

	pconn.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		offerFile, err := os.Create("offer")
		if err != nil {
			fmt.Println("Error creating offer file:", err)
			return
		}
		// defer offerFile.Close()

		_, err = offerFile.WriteString(pconn.LocalDescription().SDP)
		if err != nil {
			fmt.Println("Error writing to offer file:", err)
		}
	})

	// ICE connection state handling
	pconn.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	dataChannel, err := pconn.CreateDataChannel("data", nil)
	if err != nil {
		log.Fatal("Failed to create data channel:", err)
	}

	dataChannel.OnOpen(func() {
		fmt.Println("Data Channel Opened")
		wg.Done()
	})

	dataChannel.OnClose(func() {
		fmt.Println("Data Channel Closed")
	})

	dataChannel.OnError(func(err error) {
		fmt.Println("Data Channel Error:", err)
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Println("Data Channel Message:", string(msg.Data))
	})

	offer, err := pconn.CreateOffer(nil)
	if err != nil {
		log.Fatal("Failed to create offer:", err)
	}

	err = pconn.SetLocalDescription(offer)
	if err != nil {
		log.Fatal("Failed to set local description:", err)
	}

	fmt.Print("Enter file name: ")
	fmt.Scanln(&fileName)

	// Print the Local SDP for the user to copy and send to the client
	// localSDP := pconn.LocalDescription().SDP
	// file, err := os.Create("offer")
	// if err != nil {
	// 	log.Fatal("Error creating offer file:", err)
	// }
	// defer file.Close()

	// _, err = file.WriteString(localSDP)
	// if err != nil {
	// 	log.Fatal("Error writing to offer file:", err)
	// }

	// fmt.Println("Offer SDP written to 'offer' file")

	// Ask the user to enter the remote SDP
	fmt.Println("Please enter the remote SDP file name:")
	var remoteSDPFileName string
	fmt.Print("Enter the file name containing the remote SDP: ")
	fmt.Scanln(&remoteSDPFileName)

	remoteSDPFile, err := os.Open(remoteSDPFileName)
	if err != nil {
		log.Fatal("Error opening remote SDP file:", err)
	}
	defer remoteSDPFile.Close()

	remoteSDP, err := io.ReadAll(remoteSDPFile)
	if err != nil {
		log.Fatal("Error reading remote SDP file:", err)
	}

	// Set the remote description
	err = pconn.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  string(remoteSDP),
	})
	if err != nil {
		log.Fatal("Failed to set remote description:", err)
	}

	// Wait for the data channel to open
	fmt.Println("Waiting for data channel to open...")
	wg.Wait()

	// File transfer logic
	func() {
		file, err := os.Open(fileName)
		if err != nil {
			log.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		const chunkSize = 16 * 1024 // 16 KB chunks
		buffer := make([]byte, chunkSize)

		sendChunk := func(chunk []byte) {
			err := dataChannel.Send(chunk)
			if err != nil {
				log.Println("Error sending chunk:", err)
			}
		}

		for {
			n, err := file.Read(buffer)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Println("Error reading file:", err)
				return
			}

			chunk := buffer[:n]
			sendChunk(chunk)
		}
		fmt.Println("File sent successfully")
	}()

	// Keep the connection open for a while to ensure all data is sent
	time.Sleep(5 * time.Second)
	dataChannel.Close()
}
