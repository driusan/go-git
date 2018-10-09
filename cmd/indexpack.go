package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/driusan/dgit/git"
)

// Parses the arguments from git-unpack-objects as they were passed on the commandline
// and calls git.CatFiles
func IndexPack(c *git.Client, args []string) (err error) {
	flags := flag.NewFlagSet("index-pack", flag.ExitOnError)
	flags.SetOutput(flag.CommandLine.Output())
	flags.Usage = func() {
		flag.Usage()
		fmt.Fprintf(flag.CommandLine.Output(), "\n\nOptions:\n\n")
		flags.PrintDefaults()
	}
	options := git.IndexPackOptions{}

	flags.BoolVar(&options.Verbose, "v", false, "Print progress information to stderr")
	output := flags.String("o", "", "Write index file to output file")
	flags.BoolVar(&options.Stdin, "stdin", false, "Read the packfile from stdin and copy to packfile argument. (If packfile is unspecified, write to objects/pack directory)")
	flags.BoolVar(&options.FixThin, "fix-thin", false, "Inflate packfiles generated by git pack-objects --thin")
	flags.StringVar(&options.Keep, "keep", "", "Generate an empty .keep file. See git documentation.")
	flags.BoolVar(&options.Strict, "strict", false, "Die if the pack contains broken objects or links.")
	flags.UintVar(&options.Threads, "threads", 0, "Specify the number of threads to use to resolve deltas.")
	flags.Parse(args)
	args = flags.Args()

	// Determine where to read the pack file based on command line options.
	var packfile io.ReadSeeker
	var idx git.PackfileIndex

	if options.Stdin {
		packfile = os.Stdin
	} else if len(args) < 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "Must provide pack file name or --stdin\n")
		flags.Usage()
		os.Exit(2)
	} else {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()
		packfile = f
	}

	// Determine where to put the pack file based on command line options
	if *output != "" {
		// User specified output file
		f, err := os.Create(*output)
		if err != nil {
			return err
		}
		defer f.Close()
		options.Output = f
	} else if options.Stdin {
		if len(args) >= 1 {
			// No output file, but stdin was specified.
			f, err := os.Create(args[0])
			if err != nil {
				return err
			}
			defer f.Close()
			options.Output = f
		} else {
			// No output file or packfile was specified, but
			// --stdin was used. Save it into a file in .git/objects/pack
			// (we'll rename it in the end based on the trailer.)
			// (We use a temp file directly in the packs directory
			//  because otherwise we'll need an extra pointless copy
			//  on some operating systems, where mv can't move between
			// directories.)
			_, err := git.IndexAndCopyPack(c, options, os.Stdin)
			return err
		}
	} else {
		// Guess based on the pack name.
		if filepath.Ext(args[0]) != ".pack" {
			fmt.Fprintf(flag.CommandLine.Output(), "File name does not end in .pack\n")
			flags.Usage()
			os.Exit(2)
		}
		fname := strings.TrimSuffix(args[0], "pack") + "idx"

		f, err := os.Create(fname)
		if err != nil {
			return err
		}
		defer f.Close()
		options.Output = f
	}
	idx, err = git.IndexPack(c, options, packfile)
	if err != nil {
		return err
	}

	return idx.WriteIndex(options.Output)
}
