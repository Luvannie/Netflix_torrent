package bridge

import "os/exec"

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(name string, args ...string) (string, error) {
	output, err := exec.Command(name, args...).CombinedOutput()
	return string(output), err
}
