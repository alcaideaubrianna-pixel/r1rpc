# Repository Guidelines

## Project Structure & Module Organization

The Go service starts in `cmd/server`; `cmd/dbinit` initializes the database. Core packages live under `internal/`: `app` coordinates workflows, `rpc` manages device sessions and dispatch, `web` exposes HTTP/WebSocket endpoints, and `service`, `dao`, `store`, and `taskqueue` own business and persistence concerns. The React/TypeScript dashboard is in `web/src`; its production build is embedded from `internal/web/ui`. Deployment files live in `deploy/`, examples in `examples/`, and longer design notes in `docs/`.

Do not manually edit GoFrame-generated files in `internal/model/do`, `internal/model/entity`, or `internal/dao/internal`. Regenerate database code with `R1RPC_DAO_DSN=... make dao`, controllers with `make ctrl`, or both with `make generate`. New database access must use generated DAO/DO/Entity; do not add handwritten SQL to business services.

## Build, Test, and Development Commands

- `make dev`: start MySQL/Redis with Docker and run the Go server through Air at `http://localhost:9876`.
- `make dev-full`: build and start the complete Docker stack; use `make dev-stop` to stop it.
- `go run ./cmd/server`: run the service directly with local configuration.
- `go test ./...`: run all Go tests; `go test -race ./...` checks concurrency behavior.
- `go vet ./... && go build ./...`: perform required pre-PR validation.
- `npm run dev --prefix web`: run the dashboard development server.
- `npm run build --prefix web`: type-check and build the dashboard. Commit updated `internal/web/ui` assets.

## Coding Style & Naming Conventions

Format Go with `gofmt`; use lowercase package names without underscores and keep I/O methods context-aware. Follow the Controller -> Service -> DAO/infrastructure dependency direction described in `docs/development-standards.md`. Use TypeScript for dashboard changes, PascalCase for React components, and camelCase for functions and variables. Keep hand-written Go/TS/TSX files below 500 lines (React pages preferably below 400). Technical comments should explain why and are written in Chinese.

## Testing Guidelines

Place Go tests beside their packages as `*_test.go`, naming cases `TestXxx`. Add focused coverage for state transitions, queue behavior, persistence, and HTTP/WebSocket contracts. No numeric coverage threshold is enforced; run normal and race suites for concurrency-sensitive changes.

## Commits & Pull Requests

Use Conventional Commits, typically with Chinese summaries, such as `fix(ws): 保活与 RPC 数据流解耦`. Keep each PR focused, explain motivation and behavior changes, link relevant issues, and include screenshots for dashboard changes. Complete the PR template checks and update user-facing documentation when interfaces or behavior change.

## Security & Configuration

Copy `config.example.yaml` to an untracked `config.yaml`. Never commit secrets, production DSNs, tokens, logs, or runtime uploads. Report vulnerabilities privately according to `SECURITY.md`.
