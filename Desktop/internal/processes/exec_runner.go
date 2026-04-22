package processes

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type ExecRunner struct {
	stopTimeout time.Duration
}

func NewExecRunner(stopTimeout time.Duration) ExecRunner {
	if stopTimeout <= 0 {
		stopTimeout = 5 * time.Second
	}
	return ExecRunner{stopTimeout: stopTimeout}
}

func (r ExecRunner) Start(ctx context.Context, service Service) (Handle, error) {
	if service.Executable == "" {
		return nil, errors.New("service executable is empty")
	}

	cmdCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(cmdCtx, service.Executable, service.Args...)
	cmd.Dir = service.WorkingDir
	cmd.Env = append(os.Environ(), flattenEnv(service.Environment)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	return &execHandle{
		cancel:      cancel,
		cmd:         cmd,
		waitDone:    waitDone,
		stopTimeout: r.stopTimeout,
	}, nil
}

type execHandle struct {
	cancel      context.CancelFunc
	cmd         *exec.Cmd
	waitDone    chan error
	stopTimeout time.Duration
}

func (h *execHandle) Stop(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}

	select {
	case err := <-h.waitDone:
		if err == nil || errors.Is(err, context.Canceled) {
			return nil
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil
		}
		return err
	case <-time.After(h.stopTimeout):
		if h.cmd.Process != nil {
			_ = h.cmd.Process.Kill()
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-h.waitDone:
		if err == nil {
			return nil
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func flattenEnv(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}

	flattened := make([]string, 0, len(env))
	for key, value := range env {
		flattened = append(flattened, key+"="+value)
	}
	return flattened
}
