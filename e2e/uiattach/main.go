// Command uiattach sources a Vim script in a Neovim with a UI attached: it
// starts Neovim with --embed and attaches to it as its UI (the redraws
// ignored), for the scenarios a headless Neovim cannot run (the output of a
// terminal job of MS-Windows: issue #1).
//
//	uiattach NVIM SCRIPT
//
// The script ends Neovim (qall!); the environment is that of uiattach.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/neovim/go-client/nvim"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: uiattach NVIM SCRIPT")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "uiattach:", err)
		os.Exit(1)
	}
}

func run(bin, script string) error {
	abs, err := filepath.Abs(script)
	if err != nil {
		return err
	}
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
	done := make(chan error, 1)
	go func() { done <- v.Serve() }()
	if err := v.AttachUI(100, 30, map[string]any{"ext_linegrid": true, "rgb": true}); err != nil {
		return err
	}
	// The script quits: the answer to this may not come.
	go func() { _ = v.Command("source " + filepath.ToSlash(abs)) }()
	<-done
	return nil
}
