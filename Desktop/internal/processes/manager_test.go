package processes

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeRunner struct {
	failFor string
	started []string
	stopped []string
	handles map[string]*fakeHandle
}

type fakeHandle struct {
	name    string
	stopped *[]string
}

func (h *fakeHandle) Stop(context.Context) error {
	*h.stopped = append(*h.stopped, h.name)
	return nil
}

func (r *fakeRunner) Start(_ context.Context, service Service) (Handle, error) {
	r.started = append(r.started, service.Name)
	if service.Name == r.failFor {
		return nil, errors.New("boom")
	}
	if r.handles == nil {
		r.handles = map[string]*fakeHandle{}
	}
	handle := &fakeHandle{name: service.Name, stopped: &r.stopped}
	r.handles[service.Name] = handle
	return handle, nil
}

func TestManagerStartAllAndStopAll(t *testing.T) {
	runner := &fakeRunner{}
	manager := NewManager(runner)
	services := []Service{
		{Name: "postgres"},
		{Name: "qbittorrent"},
		{Name: "backend"},
	}

	if err := manager.StartAll(context.Background(), services); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}

	gotStates := manager.Snapshot()
	wantStates := []ProcessState{
		{Name: "postgres", Status: "running"},
		{Name: "qbittorrent", Status: "running"},
		{Name: "backend", Status: "running"},
	}
	if !reflect.DeepEqual(gotStates, wantStates) {
		t.Fatalf("Snapshot() = %#v, want %#v", gotStates, wantStates)
	}

	if err := manager.StopAll(context.Background()); err != nil {
		t.Fatalf("StopAll() error = %v", err)
	}

	if !reflect.DeepEqual(runner.stopped, []string{"backend", "qbittorrent", "postgres"}) {
		t.Fatalf("stopped order = %#v", runner.stopped)
	}
}

func TestManagerRestartRestartsNamedService(t *testing.T) {
	runner := &fakeRunner{}
	manager := NewManager(runner)
	services := []Service{
		{Name: "backend"},
	}

	if err := manager.StartAll(context.Background(), services); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}

	if err := manager.Restart(context.Background(), "backend"); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}

	if !reflect.DeepEqual(runner.started, []string{"backend", "backend"}) {
		t.Fatalf("started = %#v", runner.started)
	}
	if !reflect.DeepEqual(runner.stopped, []string{"backend"}) {
		t.Fatalf("stopped = %#v", runner.stopped)
	}
}

func TestManagerStartAllStopsStartedServicesOnFailure(t *testing.T) {
	runner := &fakeRunner{failFor: "backend"}
	manager := NewManager(runner)
	services := []Service{
		{Name: "postgres"},
		{Name: "backend"},
	}

	err := manager.StartAll(context.Background(), services)
	if err == nil {
		t.Fatalf("expected StartAll() error")
	}

	if !reflect.DeepEqual(runner.stopped, []string{"postgres"}) {
		t.Fatalf("stopped = %#v", runner.stopped)
	}

	gotStates := manager.Snapshot()
	wantStates := []ProcessState{
		{Name: "postgres", Status: "stopped"},
		{Name: "backend", Status: "failed"},
	}
	if !reflect.DeepEqual(gotStates, wantStates) {
		t.Fatalf("Snapshot() = %#v, want %#v", gotStates, wantStates)
	}
}
