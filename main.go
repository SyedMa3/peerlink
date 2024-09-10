package main

import (
	"fmt"

	peerconnection "github.com/SyedMa3/peerlink/peer-connection"
)

func main() {
	fmt.Println("Choose an option:")
	fmt.Println("1. Send a file")
	fmt.Println("2. Receive a file")

	var choice int
	fmt.Print("Enter your choice (1 or 2): ")
	_, err := fmt.Scan(&choice)
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	switch choice {
	case 1:
		peerconnection.SendFile()
	case 2:
		peerconnection.ReceiveFile()
	default:
		fmt.Println("Invalid choice. Please run the program again and select 1 or 2.")
	}
}
