package processes

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecRunnerStartAndStop(t *testing.T) {
	tempDir := t.TempDir()
	signalFile := filepath.Join(tempDir, "signal.txt")

	runner := NewExecRunner(2 * time.Second)
	handle, err := runner.Start(context.Background(), Service{
		Name:       "test-service",
		Executable: "powershell.exe",
		Args: []string{
			"-NoProfile",
			"-Command",
			"$path = $env:NETFLIX_TORRENT_SIGNAL; while (-not (Test-Path -LiteralPath $path)) { Start-Sleep -Milliseconds 100 }",
		},
		Environment: map[string]string{
			"NETFLIX_TORRENT_SIGNAL": signalFile,
		},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := os.WriteFile(signalFile, []byte("stop"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := handle.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}
