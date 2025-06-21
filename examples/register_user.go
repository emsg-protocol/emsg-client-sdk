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
		keyFile  = flag.String("key", "", "Path to private key file")
		address  = flag.String("address", "", "EMSG address to register")
		generate = flag.Bool("generate-key", false, "Generate a new key pair and save to file")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Register a user with an EMSG server.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate a new key pair\n")
		fmt.Fprintf(os.Stderr, "  %s -generate-key -key=my-key.txt\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Register a user\n")
		fmt.Fprintf(os.Stderr, "  %s -key=my-key.txt -address=alice#example.com\n\n", os.Args[0])
	}

	flag.Parse()

	if *keyFile == "" {
		fmt.Fprintf(os.Stderr, "Error: -key flag is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Generate key pair if requested
	if *generate {
		if err := generateKeyPair(*keyFile); err != nil {
			log.Fatalf("Failed to generate key pair: %v", err)
		}
		fmt.Printf("Key pair generated and saved to %s\n", *keyFile)
		return
	}

	// Validate required flags for registration
	if *address == "" {
		fmt.Fprintf(os.Stderr, "Error: -address flag is required for registration\n\n")
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

	// Register user
	fmt.Printf("Registering user %s...\n", *address)
	if err := emsgClient.RegisterUser(*address); err != nil {
		log.Fatalf("Failed to register user: %v", err)
	}

	fmt.Println("User registered successfully!")
	fmt.Printf("Public key: %s\n", keyPair.PublicKeyBase64())
}

func generateKeyPair(filename string) error {
	keyPair, err := keymgmt.GenerateKeyPair()
	if err != nil {
		return err
	}

	if err := keyPair.SavePrivateKeyToFile(filename); err != nil {
		return err
	}

	fmt.Printf("Public key: %s\n", keyPair.PublicKeyBase64())
	return nil
}
