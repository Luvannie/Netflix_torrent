package processes

import (
	"context"
	"errors"
	"sync"
)

type Service struct {
	Name        string            `json:"name"`
	Executable  string            `json:"executable"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	WorkingDir  string            `json:"workingDir,omitempty"`
}

type ProcessState struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Handle interface {
	Stop(context.Context) error
}

type Runner interface {
	Start(context.Context, Service) (Handle, error)
}

var ErrUnknownService = errors.New("unknown service")
var ErrRunnerNotConfigured = errors.New("process runner is not configured")

type managedProcess struct {
	spec   Service
	handle Handle
	status string
}

type Manager struct {
	mu       sync.Mutex
	runner   Runner
	services map[string]*managedProcess
	order    []string
}

func NewManager(runner Runner) *Manager {
	return &Manager{
		runner:   runner,
		services: map[string]*managedProcess{},
		order:    []string{},
	}
}

func (m *Manager) StartAll(ctx context.Context, services []Service) error {
	for _, service := range services {
		if err := m.start(ctx, service); err != nil {
			_ = m.StopAll(ctx)
			return err
		}
	}
	return nil
}

func (m *Manager) Restart(ctx context.Context, name string) error {
	m.mu.Lock()
	current, ok := m.services[name]
	m.mu.Unlock()
	if !ok {
		return ErrUnknownService
	}

	if current.handle != nil {
		if err := current.handle.Stop(ctx); err != nil {
			return err
		}
	}

	return m.start(ctx, current.spec)
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	order := append([]string(nil), m.order...)
	services := make(map[string]*managedProcess, len(m.services))
	for name, process := range m.services {
		services[name] = process
	}
	m.mu.Unlock()

	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		process := services[name]
		if process == nil || process.handle == nil {
			continue
		}
		if err := process.handle.Stop(ctx); err != nil {
			return err
		}

		m.mu.Lock()
		process.status = "stopped"
		process.handle = nil
		m.mu.Unlock()
	}

	return nil
}

func (m *Manager) Snapshot() []ProcessState {
	m.mu.Lock()
	defer m.mu.Unlock()

	states := make([]ProcessState, 0, len(m.order))
	for _, name := range m.order {
		process := m.services[name]
		states = append(states, ProcessState{
			Name:   name,
			Status: process.status,
		})
	}
	return states
}

func (m *Manager) start(ctx context.Context, service Service) error {
	if m.runner == nil {
		return ErrRunnerNotConfigured
	}
	handle, err := m.runner.Start(ctx, service)
	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		if existing, ok := m.services[service.Name]; ok {
			existing.status = "failed"
		} else {
			m.services[service.Name] = &managedProcess{
				spec:   service,
				status: "failed",
			}
			m.order = append(m.order, service.Name)
		}
		return err
	}

	if _, ok := m.services[service.Name]; !ok {
		m.order = append(m.order, service.Name)
	}
	m.services[service.Name] = &managedProcess{
		spec:   service,
		handle: handle,
		status: "running",
	}
	return nil
}
