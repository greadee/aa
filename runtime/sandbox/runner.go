package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// Command is one process invocation handed to a Runner.
type Command struct {
	Path  string
	Args  []string
	Dir   string
	Env   []string
	Stdin []byte
}

// Output is raw process output.
type Output struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
}

// Runner executes a single command. ExecRunner is the default; tests inject a
// deterministic fake.
type Runner interface {
	Run(ctx context.Context, cmd Command) (Output, error)
}

// ExecRunner runs the command as an OS process and terminates it when ctx is
// done.
type ExecRunner struct{}

// Run implements Runner.
func (ExecRunner) Run(ctx context.Context, cmd Command) (Output, error) {
	command := exec.CommandContext(ctx, cmd.Path, cmd.Args...)
	command.Dir = cmd.Dir
	command.Env = cmd.Env
	command.Stdin = bytes.NewReader(cmd.Stdin)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return Output{ExitCode: exitErr.ExitCode(), Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, nil
		}
		return Output{}, err
	}
	return Output{ExitCode: 0, Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, nil
}
