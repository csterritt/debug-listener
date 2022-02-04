package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

// Application constants, defining host, port, and protocol.
const (
	connectHost = "localhost"
	connectPort = "21212"
	connectType = "tcp"
)

func main() {
	// Start the client and connect to the server.
	fmt.Println("Connecting to", connectType, "server", connectHost+":"+connectPort)
	conn, err := net.Dial(connectType, connectHost+":"+connectPort)
	if err != nil {
		fmt.Println("Error connecting:", err.Error())
		os.Exit(1)
	}

	// Create new reader from Stdin.
	reader := bufio.NewReader(os.Stdin)

	if len(os.Args) > 1 {
		fmt.Println("Setting name to", os.Args[1])
		conn.Write([]byte("::name::" + os.Args[1] + "\n"))
	}

	// run loop forever, until exit.
	for {
		// Prompting message.
		fmt.Print("Text to send: ")

		// Read in input until newline, Enter key.
		input, _ := reader.ReadString('\n')

		// Send to socket connection.
		conn.Write([]byte(input))
	}
}
