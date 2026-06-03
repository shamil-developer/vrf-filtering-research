package command

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

type Result struct {
	Stdout string
	Stderr string
}

type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

func (r *Runner) Run(
	ctx context.Context,
	name string,
	args ...string,
) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}

		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Result{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			},
			fmt.Errorf("run %s: %w: %s", formatCommand(name, args), err, strings.TrimSpace(stderr.String()))
	}

	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, nil
}

func (r *Runner) RunInNamespace(
	ctx context.Context,
	namespace string,
	name string,
	args ...string,
) (Result, error) {
	commandArgs := append([]string{"netns", "exec", namespace, name}, args...)

	return r.Run(ctx, "ip", commandArgs...)
}

func (r *Runner) RunShell(
	ctx context.Context,
	command string,
) (Result, error) {
	return r.Run(ctx, "sh", "-c", command)
}

func formatCommand(
	name string,
	args []string,
) string {
	if len(args) == 0 {
		return name
	}

	return name + " " + strings.Join(args, " ")
}
