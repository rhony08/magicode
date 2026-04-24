//go:build !windows
// +build !windows

package pty

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

// PTYProcess is a Unix PTY process implementation.
type PTYProcess struct {
	cmd    *exec.Cmd
	pty    *os.File
	mu     sync.Mutex
	exited bool
	exit   *Exit
	done   chan struct{}
}

// NewProcess creates a new PTY process.
func NewProcess(command string, args []string, opts *CreateInput) (*PTYProcess, error) {
	// Prepare command
	cmd := exec.Command(command, args...)

	// Set working directory
	if opts.CWD != "" {
		cmd.Dir = opts.CWD
	}

	// Set environment
	cmd.Env = os.Environ()
	if opts.Env != nil {
		for k, v := range opts.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Add terminal-specific environment
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	cmd.Env = append(cmd.Env, "OPENCODE_TERMINAL=1")

	// Set terminal size
	size := &pty.Winsize{
		Cols: uint16(opts.Cols),
		Rows: uint16(opts.Rows),
	}
	if size.Cols == 0 {
		size.Cols = DefaultCols
	}
	if size.Rows == 0 {
		size.Rows = DefaultRows
	}

	// Start the command with a PTY
	ptmx, err := pty.StartWithSize(cmd, size)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Create process
	p := &PTYProcess{
		cmd:  cmd,
		pty:  ptmx,
		done: make(chan struct{}),
	}

	// Set process group for better signal handling
	if cmd.Process != nil {
		// On Unix, we can use the process directly
	}

	// Monitor for exit
	go p.monitorExit()

	return p, nil
}

// PID returns the process ID.
func (p *PTYProcess) PID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Write writes data to the PTY.
func (p *PTYProcess) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exited {
		return 0, fmt.Errorf("process has exited")
	}
	n, err := p.pty.Write(data)
	return n, err
}

// Read reads data from the PTY.
func (p *PTYProcess) Read(data []byte) (int, error) {
	return p.pty.Read(data)
}

// Resize changes the terminal dimensions.
func (p *PTYProcess) Resize(cols, rows int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exited {
		return fmt.Errorf("process has exited")
	}
	return pty.Setsize(p.pty, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// Kill terminates the process.
func (p *PTYProcess) Kill(signal string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exited || p.cmd.Process == nil {
		return nil
	}

	// Convert signal string to syscall signal
	var sig syscall.Signal
	switch signal {
	case "SIGTERM", "term":
		sig = syscall.SIGTERM
	case "SIGKILL", "kill":
		sig = syscall.SIGKILL
	case "SIGINT", "int":
		sig = syscall.SIGINT
	default:
		sig = syscall.SIGTERM
	}

	return p.cmd.Process.Signal(sig)
}

// Wait waits for the process to exit.
func (p *PTYProcess) Wait(ctx context.Context) (*Exit, error) {
	select {
	case <-p.done:
		return p.exit, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the PTY file handles.
func (p *PTYProcess) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error

	// Close the PTY file
	if p.pty != nil {
		if err := p.pty.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close pty: %w", err))
		}
		p.pty = nil
	}

	// Ensure process is killed if still running
	if !p.exited && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// monitorExit monitors the process for exit.
func (p *PTYProcess) monitorExit() {
	err := p.cmd.Wait()

	p.mu.Lock()
	p.exited = true

	// Determine exit code and signal
	p.exit = &Exit{}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			p.exit.Code = exitErr.ExitCode()
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					p.exit.Signal = status.Signal().String()
				}
			}
		} else {
			p.exit.Code = 1
		}
	} else {
		p.exit.Code = 0
	}

	p.mu.Unlock()
	close(p.done)
}

// GetShell returns the user's preferred shell.
func GetShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		// Default shells by platform
		shell = "/bin/bash"
	}
	return shell
}

// IsLoginShell checks if the shell needs login flag.
func IsLoginShell(shell string) bool {
	// Common shells that support -l flag
	loginShells := []string{"bash", "zsh", "sh", "dash"}
	for _, s := range loginShells {
		if shell == s || shell == "/bin/"+s || shell == "/usr/bin/"+s {
			return true
		}
	}
	return false
}

// io.Closer interface implementation
func (p *PTYProcess) Close_() error {
	return p.Close()
}

// Ensure PTYProcess implements Process interface
var _ Process = (*PTYProcess)(nil)

// Ensure PTYProcess implements io.ReadWriter
var _ interface {
	io.Reader
	io.Writer
} = (*PTYProcess)(nil)