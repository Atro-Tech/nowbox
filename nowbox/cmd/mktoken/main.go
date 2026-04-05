// mktoken generates encrypted .now files.
//
// Build with the token key:
//   go build -ldflags "-X github.com/nowbox/nowbox/internal/token.keyHex=YOUR_KEY" ./cmd/mktoken
//
// Usage:
//   ./mktoken --host sprites --agent claude SPRITES_API_KEY=sk-xxx > demo.now
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nowbox/nowbox/internal/token"
)

func main() {
	host := flag.String("host", "", "host provider")
	agent := flag.String("agent", "", "agent")
	flag.Parse()

	if *host == "" || *agent == "" {
		fmt.Fprintf(os.Stderr, "usage: mktoken --host <host> --agent <agent> KEY=VAL ...\n")
		os.Exit(1)
	}

	vars := make(map[string]string)
	for _, arg := range flag.Args() {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "mktoken: invalid var %q (expected KEY=VAL)\n", arg)
			os.Exit(1)
		}
		vars[parts[0]] = parts[1]
	}

	sealed, err := token.Seal(&token.Payload{
		Host:  *host,
		Agent: *agent,
		Vars:  vars,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "mktoken: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("#!/bin/sh\n")
	fmt.Printf("# nowbox — %s + %s\n", *host, *agent)
	fmt.Printf("set -e\n")
	fmt.Printf("export NOWBOX_TOKEN=\"%s\"\n", sealed)
	fmt.Printf("curl -fsSL nowbox.lol | sh -s -- %s %s\n", *host, *agent)
}
