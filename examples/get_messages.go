package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/emsg-protocol/emsg-client-sdk/client"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func main() {
	var (
		keyFile = flag.String("key", "", "Path to private key file")
		address = flag.String("address", "", "EMSG address to get messages for")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Retrieve messages for an EMSG address.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -key=my-key.txt -address=alice#example.com\n\n", os.Args[0])
	}

	flag.Parse()

	if *keyFile == "" || *address == "" {
		fmt.Fprintf(os.Stderr, "Error: -key and -address flags are required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load key pair
	keyPair, err := keymgmt.LoadPrivateKeyFromFile(*keyFile)
	if err != nil {
		log.Fatalf("Failed to load key pair: %v", err)
	}

	// Create client
	emsgClient := client.NewWithKeyPair(keyPair)

	// Get messages
	fmt.Printf("Retrieving messages for %s...\n", *address)
	messages, err := emsgClient.GetMessages(*address)
	if err != nil {
		log.Fatalf("Failed to get messages: %v", err)
	}

	if len(messages) == 0 {
		fmt.Println("No messages found.")
		return
	}

	fmt.Printf("Found %d message(s):\n\n", len(messages))

	for i, msg := range messages {
		fmt.Printf("Message %d:\n", i+1)
		fmt.Printf("  From: %s\n", msg.From)
		fmt.Printf("  To: %v\n", msg.To)
		if len(msg.CC) > 0 {
			fmt.Printf("  CC: %v\n", msg.CC)
		}
		if msg.Subject != "" {
			fmt.Printf("  Subject: %s\n", msg.Subject)
		}
		fmt.Printf("  Body: %s\n", msg.Body)
		if msg.GroupID != "" {
			fmt.Printf("  Group ID: %s\n", msg.GroupID)
		}
		fmt.Printf("  Timestamp: %d\n", msg.Timestamp)
		fmt.Printf("  Message ID: %s\n", msg.MessageID)
		fmt.Printf("  Signed: %t\n", msg.IsSigned())
		fmt.Println()
	}
}
