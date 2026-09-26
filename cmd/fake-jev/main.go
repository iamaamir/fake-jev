// Command fake-jev is the deterministic, zero-model fake server that
// implements a strict jev/v1 compatibility profile over HTTP + JSON.
package main

import (
	"os"

	"fake-jev/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
