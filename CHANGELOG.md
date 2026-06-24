# Changelog

All notable changes to this project are documented here.
This project follows [Semantic Versioning](https://semver.org/) (`vMAJOR.MINOR.PATCH`).

## v1.0.0 — 2026-06-24

First stable release under standard versioning after the CLI restructure.

### Added
- Standard Go CLI layout (`cmd/layrgo`, `internal/`)
- Interactive and non-interactive `layrgo init`
- `layrgo version` command
- Compilable generated projects with DI wiring
- Docker and docker-compose per database
- CI workflow (`go test`, `go vet`)
- Pinned dependency versions for generated projects

### Supported stacks
- Frameworks: Fiber, Gin, Echo, Chi, Iris
- Databases: Postgres, MySQL, SQLite
- ORMs: GORM, XORM, Ent, or raw `database/sql`

[Unreleased]: https://github.com/samznd/layrgo/compare/v1.0.0...HEAD
