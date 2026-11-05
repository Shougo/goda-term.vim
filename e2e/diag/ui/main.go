// Command ui sources a script in a Neovim with a UI attached (issue #1): it
// starts Neovim with --embed and attaches to it as its UI, the redraws
// ignored.
//
//	go run . NVIM SCRIPT
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/neovim/go-client/nvim"
)

func main() {
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "ui:", err)
		os.Exit(1)
	}
}

func run(bin, script string) error {
	v, err := nvim.NewChildProcess(
		nvim.ChildProcessCommand(bin),
		nvim.ChildProcessArgs("--embed", "--clean", "-n"),
		nvim.ChildProcessServe(false),
	)
	if err != nil {
		return err
	}
	defer v.Close()
	if err := v.RegisterHandler("redraw", func(...[]any) {}); err != nil {
		return err
	}
	go func() { _ = v.Serve() }()
	if err := v.AttachUI(100, 30, map[string]any{"ext_linegrid": true, "rgb": true}); err != nil {
		return err
	}
	out, err := filepath.Abs(os.Getenv("DIAG_OUT"))
	if err != nil {
		return err
	}
	if err := v.SetVar("diag_out", out); err != nil {
		return err
	}
	if err := v.Command("let $DIAG_MODE = 'ui' | let $DIAG_OUT = g:diag_out"); err != nil {
		return err
	}
	abs, err := filepath.Abs(script)
	if err != nil {
		return err
	}
	if err := v.Command("source " + filepath.ToSlash(abs)); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	_ = v.Command("qall!") // the channel goes with Neovim
	return nil
}
