//go:build linux || darwin

package pty

import (
	"errors"
	"os"
	"syscall"
	"time"
	"unsafe"
)

func start(p *Pty, o Options) error {
	master, slave, err := open()
	if err != nil {
		return err
	}
	ws := struct{ row, col, x, y uint16 }{uint16(o.Rows), uint16(o.Cols), 0, 0}
	_ = ioctl(slave.Fd(), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws)))
	p.cmd.Stdin, p.cmd.Stdout, p.cmd.Stderr = slave, slave, slave
	// Its own session, the terminal as its controlling terminal (fd 0).
	p.cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := p.cmd.Start(); err != nil {
		master.Close()
		slave.Close()
		return err
	}
	// The command has it now.
	slave.Close()
	p.r, p.w = master, master
	return nil
}

func ioctl(fd, req, arg uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg); e != 0 {
		return e
	}
	return nil
}

func closeTTY(*Pty) {}

// hangup: SIGHUP to the session of the command (as when its terminal is
// gone), and SIGKILL after a moment.
func hangup(p *Pty) {
	if p.cmd.Process == nil {
		return
	}
	pid := p.cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGHUP)
	select {
	case <-p.done:
	case <-time.After(500 * time.Millisecond):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
}

// Interrupt sends SIGINT to the command (as <C-c> in its terminal).
func (p *Pty) Interrupt() error {
	if p.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-p.cmd.Process.Pid, syscall.SIGINT)
}

// Resize sets the size of the terminal.
func (p *Pty) Resize(rows, cols int) error {
	f, ok := p.r.(*os.File)
	if !ok {
		return nil
	}
	ws := struct{ row, col, x, y uint16 }{uint16(rows), uint16(cols), 0, 0}
	return ioctl(f.Fd(), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws)))
}

func isEIO(err error) bool { return errors.Is(err, syscall.EIO) }
