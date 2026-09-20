package remote

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func Run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected clone, push, pull, or remote")
	}
	command := args[0]
	flags := flag.NewFlagSet("ssc "+command, flag.ContinueOnError)
	flags.SetOutput(out)
	var branch, override string
	if command == "clone" {
		flags.StringVar(&branch, "branch", "", "branch to check out (defaults to main, master, or first branch)")
	}
	if command == "push" || command == "pull" {
		flags.StringVar(&override, "remote", "", "repository URL override for this operation")
	}
	flags.Usage = func() {
		switch command {
		case "clone":
			fmt.Fprintln(out, "Usage: ssc clone [--branch <name>] <repository-url> <new-directory>")
		case "remote":
			fmt.Fprintln(out, "Usage: ssc remote [repository-url]")
		default:
			fmt.Fprintf(out, "Usage: ssc %s [--remote <repository-url>]\n", command)
		}
		flags.PrintDefaults()
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if command == "clone" {
		if flags.NArg() != 2 {
			return fmt.Errorf("usage: ssc clone [--branch <name>] <repository-url> <new-directory>")
		}
		c, err := NewClient(flags.Arg(0), os.Getenv("SSC_TOKEN"))
		if err != nil {
			return err
		}
		if err := Clone(ctx, c, flags.Arg(1), branch); err != nil {
			return err
		}
		fmt.Fprintln(out, "Clone complete.")
		return nil
	}
	r, err := Open(".")
	if err != nil {
		return err
	}
	if command == "remote" {
		if flags.NArg() > 1 {
			return fmt.Errorf("usage: ssc remote [repository-url]")
		}
		if flags.NArg() == 1 {
			return r.SetRemote(flags.Arg(0))
		}
		raw, err := r.Remote()
		if err != nil {
			return err
		}
		fmt.Fprintln(out, raw)
		return nil
	}
	if command != "push" && command != "pull" {
		return fmt.Errorf("unknown remote command")
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments; put flags before positional arguments")
	}
	if override == "" {
		override, err = r.Remote()
		if err != nil {
			return err
		}
	}
	c, err := NewClient(override, os.Getenv("SSC_TOKEN"))
	if err != nil {
		return err
	}
	if command == "push" {
		err = Push(ctx, r, c)
	} else {
		err = Pull(ctx, r, c)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s complete.\n", command)
	return nil
}
