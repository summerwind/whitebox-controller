package main

import (
	"fmt"
	"os"
)

var usage = `usage: whitebox-gen <command> [<args>]

Commands:
  manifest Generate manifest based on config file
`

func main() {
	var err error

	if len(os.Args) <= 1 {
		fmt.Print(usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "manifest":
		err = manifest(os.Args[2:])
	case "token":
		err = token(os.Args[2:])
	default:
		fmt.Print(usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
