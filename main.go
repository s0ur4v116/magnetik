package main

import (
	"encoding/json"
	"fmt"
	"io"
	"magnetik/utils/bencode"
	"os"
	"strings"
)

func printUsage() {
	usage := `Usage:
  decode   Decode a bencoded string
`
	fmt.Fprintf(os.Stderr, "%s", usage)
}

func decodeAndPrint(inputStr string) error {
	result, err := bencode.Decode(strings.NewReader(inputStr))
	if err != nil {
		return fmt.Errorf("Error decoding bencode: %v\n", err)
	}

	jsonBytes, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return fmt.Errorf("Error formatting output: %v\n", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

func main() {
	args := os.Args

	if len(args) <= 1 {
		printUsage()
		os.Exit(1)
	}

	switch args[1] {
	case "decode":
		var inputStr string

		if len(args) == 3 {
			inputStr = args[2]
		} else {
			bytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				os.Exit(1)
			}
			inputStr = string(bytes)
		}

		if err := decodeAndPrint(inputStr); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}
}
