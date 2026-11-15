// Command bulkcase makes a made-up case of many devices, to see how long
// the steps take and how the window behaves with a large case (spec 16 and
// 19). It is a tool for developing: it is not part of what the users get.
//
//	go run ./cmd/bulkcase <folder> [-devices 300] [-lines 500] [-sides new,old]
//
// The folder becomes a case folder with the configurations in input/new
// (and input/old). Every value is made up (package bulk).
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kojio145/configchief/internal/bulk"
	"github.com/kojio145/configchief/internal/project"
)

func main() {
	fs := flag.NewFlagSet("bulkcase", flag.ExitOnError)
	devices := fs.Int("devices", 300, "the number of devices")
	lines := fs.Int("lines", 500, "the lines of one configuration")
	sides := fs.String("sides", "new", "where the configurations go: new, old or new,old")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: bulkcase <folder> [-devices 300] [-lines 500] [-sides new,old]")
		fs.PrintDefaults()
	}
	// The folder may come before or after the options.
	args := os.Args[1:]
	dir := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dir, args = args[0], args[1:]
	}
	_ = fs.Parse(args)
	if dir == "" && fs.NArg() == 1 {
		dir = fs.Arg(0)
	} else if fs.NArg() != 0 {
		dir = ""
	}
	if dir == "" {
		fs.Usage()
		os.Exit(2)
	}
	if err := run(dir, *devices, *lines, strings.Split(*sides, ",")); err != nil {
		fmt.Fprintln(os.Stderr, "bulkcase:", err)
		os.Exit(1)
	}
}

func run(dir string, devices, lines int, sides []string) error {
	for _, s := range sides {
		if s != "new" && s != "old" {
			return fmt.Errorf("-sides takes new, old or new,old, not %q", s)
		}
	}
	files, err := bulk.Configs(devices, lines)
	if err != nil {
		return err
	}
	if left, err := os.ReadDir(dir); err == nil && len(left) > 0 {
		return fmt.Errorf("%s is not empty", dir)
	}
	p, err := project.Init(dir, fmt.Sprintf("bulk %d", devices), "")
	if err != nil {
		return err
	}
	for _, f := range files {
		for _, s := range sides {
			if err := os.WriteFile(p.Path("input/"+s+"/"+f.Name), []byte(f.Text), 0o644); err != nil {
				return err
			}
		}
	}
	fmt.Printf("%s: %d devices of %d lines in input/%s\n", dir, len(files), lines, strings.Join(sides, " and input/"))
	fmt.Println("next: import --source new, book --from new, generate (each with --project " + dir + ")")
	return nil
}
