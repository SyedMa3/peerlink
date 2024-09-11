package peerconnection

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/pion/webrtc/v3"
)

func ReceiveFile() {

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun1.l.google.com:19302",
					"stun:stun2.l.google.com:19302",
					"stun:stun3.l.google.com:19302",
				},
			},
		},
	}

	pconn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := pconn.Close(); err != nil {
			panic(err)
		}
	}()
	gatherDone := webrtc.GatheringCompletePromise(pconn)

	wg := sync.WaitGroup{}
	wg.Add(1)

	// pconn.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
	// 	fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	// 	// fmt.Println(pconn.LocalDescription().SDP)
	// })

	pconn.OnDataChannel(func(dc *webrtc.DataChannel) {
		fmt.Printf("Data channel %s %d opened\n", dc.Label(), dc.ID())

		dc.OnOpen(func() {
			fmt.Println("Data channel opened")
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Received message: %s\n", string(msg.Data))
		})

		dc.OnClose(func() {
			fmt.Println("Data channel closed")
			wg.Done()
		})

	})

	fmt.Print("Enter the file name containing the SDP of the client: ")
	var fileName string
	fmt.Scanln(&fileName)

	file, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	clientSDP, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	offer := webrtc.SessionDescription{
		SDP:  string(clientSDP),
		Type: webrtc.SDPTypeOffer,
	}
	// _, err = offer.Unmarshal()
	// if err != nil {
	// 	panic(err)
	// }

	err = pconn.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	answer, err := pconn.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	err = pconn.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	<-gatherDone
	WriteSDPToFile(pconn, "answer")

	// // Write the answer SDP to a file
	// answerFile, err := os.Create("answer")
	// if err != nil {
	// 	panic(err)
	// }
	// defer answerFile.Close()

	// _, err = answerFile.WriteString(jsonifySDP(pconn.LocalDescription().SDP))
	// if err != nil {
	// 	fmt.Println("Error writing to answer file:", err)
	// }
	// fmt.Println("Answer written to file")

	// fmt.Println("Local SDP:")
	wg.Wait()

}
