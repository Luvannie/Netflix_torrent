package bridge

import (
	"errors"
	"reflect"
	"testing"
)

type fakeCommandRunner struct {
	commands [][]string
	fail     error
	output   string
}

func (r *fakeCommandRunner) Run(name string, args ...string) (string, error) {
	command := append([]string{name}, args...)
	r.commands = append(r.commands, command)
	return r.output, r.fail
}

func TestChooseDirectoryUsesNativePickerScript(t *testing.T) {
	runner := &fakeCommandRunner{}
	native := NewNativeBridge(runner)

	_, err := native.ChooseDirectory()
	if err != nil {
		t.Fatalf("ChooseDirectory() error = %v", err)
	}

	if len(runner.commands) != 1 {
		t.Fatalf("commands = %d", len(runner.commands))
	}
	if runner.commands[0][0] != "powershell.exe" {
		t.Fatalf("command = %#v", runner.commands[0])
	}
	if path, _ := native.ChooseDirectory(); path != "" {
		t.Fatalf("expected empty path on second call, got %q", path)
	}
}

func TestChooseDirectoryReturnsSelectedPath(t *testing.T) {
	runner := &fakeCommandRunner{output: "C:\\Media\\Movies\r\n"}
	native := NewNativeBridge(runner)

	path, err := native.ChooseDirectory()
	if err != nil {
		t.Fatalf("ChooseDirectory() error = %v", err)
	}
	if path != `C:\Media\Movies` {
		t.Fatalf("path = %q", path)
	}
}

func TestOpenPathUsesExplorer(t *testing.T) {
	runner := &fakeCommandRunner{}
	native := NewNativeBridge(runner)

	if err := native.OpenPath(`C:\logs`); err != nil {
		t.Fatalf("OpenPath() error = %v", err)
	}

	want := [][]string{{"explorer.exe", `C:\logs`}}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
}

func TestOpenPathReturnsRunnerError(t *testing.T) {
	runner := &fakeCommandRunner{fail: errors.New("boom")}
	native := NewNativeBridge(runner)

	err := native.OpenPath(`C:\logs`)
	if err == nil {
		t.Fatalf("expected error")
	}
}
