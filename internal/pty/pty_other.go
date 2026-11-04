//go:build !linux && !darwin

package pty

import (
	"io"
	"os"
)

// Without a pseudo terminal: pipes (no colors from most commands, and no
// terminal for those that ask for one).
func start(p *Pty, _ Options) error {
	in, err := p.cmd.StdinPipe()
	if err != nil {
		return err
	}
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	p.cmd.Stdout, p.cmd.Stderr = w, w
	if err := p.cmd.Start(); err != nil {
		r.Close()
		w.Close()
		return err
	}
	p.r, p.w, p.wpipe = r, in, w
	return nil
}

// closeTTY: the command is gone, so is the writing end of the pipe (read
// ends then).
func closeTTY(p *Pty) {
	if c, ok := p.wpipe.(io.Closer); ok {
		c.Close()
	}
}

func hangup(*Pty) {}

// Interrupt ends the command (there is no terminal for a signal).
func (p *Pty) Interrupt() error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *Pty) Resize(int, int) error { return nil }

func isEIO(error) bool { return false }
