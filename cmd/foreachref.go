package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/driusan/dgit/git"
)

func ForEachRef(c *git.Client, args []string) error {
	flags := flag.NewFlagSet("for-each-ref", flag.ExitOnError)
	flags.SetOutput(flag.CommandLine.Output())
	flags.Usage = func() {
		flag.Usage()
		fmt.Fprintf(flag.CommandLine.Output(), "\n\nOptions:\n")
		flags.PrintDefaults()
	}

	// These flags can be moved out of these lists and below as proper flags as they are implemented
	for _, sf := range []string{"format"} {
		flags.Var(newNotimplStringValue(), sf, "Not implemented")
	}

	flags.Parse(args)
	refs, err := git.ShowRef(c, git.ShowRefOptions{}, []string{})
	if err != nil {
		return err
	}

	patterns := flags.Args()

	for _, ref := range refs {
		if len(patterns) == 0 {
			fmt.Printf("%s %s\t%s\n", ref.Value, "commit", ref.Name) // FIXME check if commit or other object type
		} else {
			for _, pattern := range patterns {
				if strings.HasPrefix(ref.Name, pattern) { // FIXME support actual patterns and match only on path segments
					fmt.Printf("%s %s\t%s\n", ref.Value, "commit", ref.Name) // FIXME check if commit or other object type
					break
				}
			}
		}
	}

	if len(refs) == 0 {
		os.Exit(1)
	}
	return nil
}
