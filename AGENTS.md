# Agent Instructions for dbConnector

This repository provides `dbc`, a JSON-first CLI for AI agents to inspect and operate MySQL and Redis safely.

Primary agent-facing documentation:

- [docs/agent-usage.md](docs/agent-usage.md)

Quick rules:

- Prefer `./bin/dbc` when it exists.
- All command output is JSON.
- Parse `ok` before reading result fields.
- Treat `ok: false` as a command failure even if the process output is valid JSON.
- Use read-only commands by default.
- Do not run write commands unless the user explicitly asks for a write operation.
- Write commands require `--write` and may still be blocked by config.
- Never print or expose DSN, password, token, or secret values.
- Use `./bin/dbc profile list` to discover available profiles.

