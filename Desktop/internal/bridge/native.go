package bridge

import (
	"fmt"
	"strings"
)

type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

type NativeBridge struct {
	runner CommandRunner
}

func NewNativeBridge(runner CommandRunner) NativeBridge {
	return NativeBridge{runner: runner}
}

func (b NativeBridge) ChooseDirectory() (string, error) {
	if b.runner == nil {
		return "", fmt.Errorf("native command runner is not configured")
	}

	script := "Add-Type -AssemblyName System.Windows.Forms; $dialog = New-Object System.Windows.Forms.FolderBrowserDialog; if($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK){ Write-Output $dialog.SelectedPath }"
	output, err := b.runner.Run("powershell.exe", "-NoProfile", "-STA", "-Command", script)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (b NativeBridge) OpenPath(path string) error {
	if b.runner == nil {
		return fmt.Errorf("native command runner is not configured")
	}
	_, err := b.runner.Run("explorer.exe", path)
	return err
}
