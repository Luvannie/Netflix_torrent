# Desktop Shell

Milestone 1 builds the native runtime core that the future Wails shell will host.

Current scope:

- persisted launcher config
- single-instance lock
- process supervision contracts for backend and sidecars
- bootstrap state and diagnostics snapshots
- reverse proxy contract for backend HTTP traffic

The full Wails runtime wiring remains a follow-up once the Desktop runtime contracts are stable.
