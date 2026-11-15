// Command configchief runs the command line of ConfigChief without the
// window: for "go run" while developing. What the users get is one program,
// ConfigChief.exe, which runs the same commands (spec 15).
package main

import (
	"os"

	"github.com/kojio145/configchief/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
