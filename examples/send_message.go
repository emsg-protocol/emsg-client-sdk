package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/emsg-protocol/emsg-client-sdk/client"
	"github.com/emsg-protocol/emsg-client-sdk/keymgmt"
)

func main() {
	var (
		keyFile   = flag.String("key", "", "Path to private key file")
		from      = flag.String("from", "", "Sender EMSG address")
		to        = flag.String("to", "", "Recipient EMSG address(es), comma-separated")
		cc        = flag.String("cc", "", "CC EMSG address(es), comma-separated (optional)")
		subject   = flag.String("subject", "", "Message subject (optional)")
		body      = flag.String("body", "", "Message body")
		groupID   = flag.String("group", "", "Group ID for group messages (optional)")
		generate  = flag.Bool("generate-key", false, "Generate a new key pair and save to file")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Send an EMSG message using the EMSG client SDK.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate a new key pair\n")
		fmt.Fprintf(os.Stderr, "  %s -generate-key -key=my-key.txt\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Send a message\n")
		fmt.Fprintf(os.Stderr, "  %s -key=my-key.txt -from=alice#example.com -to=bob#example.org -subject=\"Hello\" -body=\"Hello, Bob!\"\n\n", os.Args[0])
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

	// Validate required flags for sending message
	if *from == "" || *to == "" || *body == "" {
		fmt.Fprintf(os.Stderr, "Error: -from, -to, and -body flags are required for sending messages\n\n")
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

	// Parse recipients
	toAddresses := parseAddressList(*to)
	var ccAddresses []string
	if *cc != "" {
		ccAddresses = parseAddressList(*cc)
	}

	// Compose message
	msgBuilder := emsgClient.ComposeMessage().
		From(*from).
		To(toAddresses...).
		Body(*body)

	if len(ccAddresses) > 0 {
		msgBuilder = msgBuilder.CC(ccAddresses...)
	}

	if *subject != "" {
		msgBuilder = msgBuilder.Subject(*subject)
	}

	if *groupID != "" {
		msgBuilder = msgBuilder.GroupID(*groupID)
	}

	msg, err := msgBuilder.Build()
	if err != nil {
		log.Fatalf("Failed to build message: %v", err)
	}

	// Send message
	fmt.Printf("Sending message from %s to %s...\n", *from, *to)
	if err := emsgClient.SendMessage(msg); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Println("Message sent successfully!")
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

func parseAddressList(addresses string) []string {
	var result []string
	for _, addr := range strings.Split(addresses, ",") {
		addr = strings.TrimSpace(addr)
		if addr != "" {
			result = append(result, addr)
		}
	}
	return result
}
