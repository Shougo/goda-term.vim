// Package pty runs a command in a pseudo terminal (Linux, macOS), or with
// pipes elsewhere (MS-Windows, for now): what it writes, and what is
// written to it.
package pty

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
)

// Pty is a command running in a pseudo terminal.
type Pty struct {
	cmd *exec.Cmd
	// r is what the command writes (stdout and stderr), w its input.
	r io.ReadCloser
	w io.WriteCloser
	// wpipe is the writing end of the output pipe (without a terminal).
	wpipe any
	once  sync.Once
	done  chan struct{}
	err   error
}

// Options are those of Start.
type Options struct {
	Args []string
	// Env is the whole environment ("NAME=value").
	Env []string
	Dir string
	// Rows and Cols are the size of the terminal (default 24x80).
	Rows, Cols int
}

// Start runs path (an executable, with its directory) with o.
func Start(path string, o Options) (*Pty, error) {
	if o.Rows <= 0 {
		o.Rows = 24
	}
	if o.Cols <= 0 {
		o.Cols = 80
	}
	cmd := exec.Command(path, o.Args...)
	cmd.Env = o.Env
	cmd.Dir = o.Dir
	p := &Pty{cmd: cmd, done: make(chan struct{})}
	if err := start(p, o); err != nil {
		return nil, err
	}
	go func() {
		p.err = cmd.Wait()
		closeTTY(p)
		close(p.done)
	}()
	return p, nil
}

// Read reads what the command wrote; io.EOF once it exited and all is
// read.
func (p *Pty) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if err != nil && isEOF(err) {
		err = io.EOF
	}
	return n, err
}

// Write types b in the command.
func (p *Pty) Write(b []byte) (int, error) { return p.w.Write(b) }

// Wait waits for the command, and returns its exit code (-1: killed).
func (p *Pty) Wait() int {
	<-p.done
	var ee *exec.ExitError
	if errors.As(p.err, &ee) {
		return ee.ExitCode()
	}
	if p.err != nil {
		return -1
	}
	return 0
}

// Close ends the command (as a hangup of its terminal) and releases the
// terminal.
func (p *Pty) Close() error {
	p.once.Do(func() {
		hangup(p)
		select {
		case <-p.done:
		default:
			if p.cmd.Process != nil {
				_ = p.cmd.Process.Kill()
			}
		}
		_ = p.w.Close()
		_ = p.r.Close()
	})
	return nil
}

// Pid is that of the command.
func (p *Pty) Pid() int {
	if p.cmd.Process == nil {
		return -1
	}
	return p.cmd.Process.Pid
}

func isEOF(err error) bool {
	// Linux reads EIO from the terminal once the command and its children
	// are gone.
	return errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) || isEIO(err)
}
