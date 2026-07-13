# Repository Guidelines

## Project Structure & Module Organization

This repository contains a Go REST API and a small React API-testing client.

- `cmd/server/` contains the API entry point.
- `internal/` holds domain features and infrastructure (handlers, services, repositories, email, messaging, and storage). Keep feature code grouped by responsibility, for example `internal/category/{domain,dto,repository,service,handler}`.
- `pkg/` contains reusable packages such as pagination; `docs/` contains generated Swagger/OpenAPI files.
- `frontend/` is the Vite, React, TypeScript client; UI sections live in `frontend/src/sections/`.
- `k8s/` contains numbered Kubernetes manifests; `docker-compose.yml` defines the local multi-service environment.
- `third_party/walk/` is vendored third-party code. Do not modify it unless intentionally updating that dependency.

## Build, Test, and Development Commands

- `docker-compose up --build` starts the full local stack (API, MongoDB, Redis, MinIO, RabbitMQ, and frontend where configured). Copy `.env.example` to `.env` first and supply local values.
- `go run ./cmd/server` runs the API directly; required service settings come from environment variables.
- `go test ./...` runs all Go tests.
- `gofmt -w <changed-go-files>` formats Go changes before committing.
- `cd frontend && npm install && npm run dev` starts the Vite client; `npm run build` type-checks and builds it.
- `kubectl apply -f k8s/00-namespace.yaml -f k8s/01-mongodb/ ...` deploys to Kubernetes; follow the complete ordered command in `README.md`.

## Coding Style & Naming Conventions

Use idiomatic Go: tabs, `gofmt`, exported PascalCase identifiers, and concise lowercase package names. Keep HTTP concerns in `handler`, business logic in `service`, and persistence in `repository`. Use `*_test.go` for Go tests. In TypeScript, follow the existing two-space indentation, PascalCase React component filenames (for example `ProfileSection.tsx`), and camelCase functions and variables.

## Testing Guidelines

Add focused unit tests next to the package under test and name cases `Test<Behavior>`. Exercise success and error paths, especially authentication, storage, messaging, and health checks. Run `go test ./...` and `cd frontend && npm run build` before opening a pull request; there is no configured coverage threshold or frontend test runner.

## Commit & Pull Request Guidelines

Recent commits use short, imperative Russian summaries, often prefixed by the lab scope (for example `ЛР9: ...` or `SMTP: ...`). Preserve that concise style and keep each commit narrowly focused. Pull requests should explain the behavioral change, list validation commands, link the relevant lab/task or issue, and include screenshots for visible frontend or Swagger changes. Never commit `.env` files or real credentials; update deployment secrets through the documented local/Kubernetes configuration process.
