// Command who is the CLI entry point for Sophia Who?,
// the holon identity manager.
package main

import (
	"fmt"
	"os"

	"github.com/organic-programming/sophia-who/internal/cli"
	"github.com/organic-programming/sophia-who/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "new":
		err = cli.RunNew()
	case "show":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: who show <uuid>")
			os.Exit(1)
		}
		err = cli.RunShow(os.Args[2])
	case "list":
		if len(os.Args) > 3 {
			fmt.Fprintln(os.Stderr, "usage: who list [root]")
			os.Exit(1)
		}
		root := "."
		if len(os.Args) == 3 {
			root = os.Args[2]
		}
		err = cli.RunList(root)
	case "serve":
		listenURI := "tcp://:9090"
		for i, arg := range os.Args[2:] {
			if arg == "--listen" && i+1 < len(os.Args[2:]) {
				listenURI = os.Args[2+i+1]
			}
			// Backward compatibility: --port 9090 → tcp://:9090
			if arg == "--port" && i+1 < len(os.Args[2:]) {
				listenURI = "tcp://:" + os.Args[2+i+1]
			}
		}
		err = server.ListenAndServe(listenURI, true)
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Sophia Who? — holon identity manager

Usage:
  who new                                     create a new holon identity
  who show <uuid>                             display a holon's identity
  who list [root]                             list all known holons in root
  who serve [--listen tcp://:9090]            start gRPC server
  who serve --listen unix:///tmp/who.sock     Unix domain socket
  who serve --listen stdio://                 stdin/stdout pipe`)
}
