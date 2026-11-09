# Changelog

All notable changes to this project are documented here.
This project follows [Semantic Versioning](https://semver.org/) (`vMAJOR.MINOR.PATCH`).

## v1.1.0 — 2026-11-09

Correctness release plus an internal restructure. Fixes three bugs in v1.0.0 that
could lose work or produce a project that does not run, hardens the generated code,
and moves template generation onto `text/template`.

### Fixed
- **`init` no longer overwrites an existing project.** Scaffolding into a non-empty
  directory is refused; pass `--force` to override. Previously a mistyped project name
  truncated `main.go`, `config/database.go` and the handler in place while `go.mod`
  kept the old dependencies, leaving the project unbuildable.
- **`docker-compose.yml` now matches the database for lowercase flag values.**
  `--database sqlite` or `--database mysql` silently generated a Postgres service,
  because the compose template compared the database name case-sensitively while input
  was validated case-insensitively. Framework, database and ORM values are now
  normalized to their canonical spelling once, at the input boundary.
- **Generated projects start without a `.env` file.** `InitialEnv()` treated a missing
  `.env` as fatal, so `docker compose up --build` — the command the CLI itself prints —
  failed for every generated project: the image ships only the binary and supplies
  configuration as environment variables. A missing `.env` is now ignored, and real
  environment variables take precedence as before.
- **Generated projects no longer fail to install on Go 1.23.** `go mod tidy` ran after
  `go get` and re-resolved pinned modules to their newest compatible release, selecting
  `gorm.io/driver/postgres` v1.6.3 — a module requiring Go 1.25+. With
  `GOTOOLCHAIN=local` the install aborted, so `--database postgres --orm gorm` could
  not be scaffolded at all. Tidy is now advisory and the pins are re-asserted
  afterwards, so the versions the README promises are the versions you get.
- **Generated handlers no longer return internal error text to clients.** Handlers
  responded with `err.Error()`, leaking driver and query detail; they now return a
  generic message.
- **Invalid project names are rejected before any files are written.** Names containing
  path separators (`../escaped` wrote outside the working directory) or characters
  invalid in a Go module path (`my app`) previously failed only after the directory and
  roughly ten files had been created, leaving debris behind.
- A failed scaffold removes the directory it created instead of leaving a partially
  generated project behind. Pre-existing directories are never removed.

### Added
- `--force` flag on `layrgo init`, to scaffold into a non-empty directory.
- Generated projects include a `.gitignore` (ignoring `.env`, build output and `*.db`)
  and a committed `.env.example`, so the default path no longer leads to committing
  database credentials.
- `scaffold.WriteFiles`, which generates project files without installing
  dependencies, allowing every stack combination to be tested without network access.

### Changed
- **Generated `config.Connect()` returns an `error` instead of calling `log.Fatal`,**
  and a matching `config.Close()` was added. `cmd/main.go` handles the error and defers
  the close, which makes the database layer usable from tests and from callers that
  want to retry. *Regenerating an existing project requires updating any custom caller
  of `config.Connect()`.*
- Generated database setup configures connection-pool limits (`SetMaxOpenConns`,
  `SetMaxIdleConns`, `SetConnMaxLifetime`, `SetConnMaxIdleTime`) rather than leaving
  the pool unbounded.
- `context.Context` is threaded through the generated layers: handlers pass the request
  context to the service, which passes it to the repository, which uses the `...Context`
  query variants. Database work is now cancelled when a request is.
- Generated files are written in a deterministic order, so progress output and any
  failure are reproducible between runs.

### Internal
- Templates moved from `fmt.Sprintf` string literals to `text/template` files embedded
  with `//go:embed` (`internal/templates/files/`). Named fields replace repeated
  positional `%s` arguments — a template referencing an unknown field now fails at
  generation time instead of silently producing misordered imports. Adding a framework
  is a template file plus one map entry.
- Per-database driver, DSN and dialector details are consolidated into a single
  `dbSpecs` table, replacing nine near-identical config literals.
- Removed `internal/scaffold/fs.go` and `internal/fileutil.dirOf`, which reimplemented
  `filepath.Join`, `os.MkdirAll` and `filepath.Dir`; `dirOf` also mishandled mixed path
  separators.
- `fileutil.CreateFile` no longer prints to stdout. Progress reporting moved to the
  caller and is redirectable via `scaffold.SetProgressOutput`, so tests run quietly.
- Unsupported framework/database/ORM combinations return a descriptive error instead of
  an empty template string.

### Testing
- Table-driven coverage of all 60 framework x database x ORM combinations.
- An integration test that builds a generated project, runs the binary with **no**
  `.env` present, and asserts `/api/v1/message` returns the expected payload.
- `TestRepositoryTemplate_XORM_ErrorPath` was tautological — its condition could never
  be true, so it passed regardless of the template's contents. Replaced with assertions
  on the specific error paths.
- Added tests for `internal/fileutil`, which previously had none.
- Regression tests for the directory guard, name validation and case normalization.

### Repo
- Removed the 6.4 MB `goscaf` binary from version control (the file is untouched on
  disk) and extended `.gitignore` to cover build output.
- Pinned the Go version in the release workflow, matching CI, so releases are
  reproducible.

### Upgrade notes
Projects generated with v1.0.0 keep working as they are. Regenerating with v1.1.0
changes the `config` package's API (`Connect() error` plus `Close()`), so update any
custom caller. To pick up only the `.env` fix in an existing project, allow a missing
`.env` in `pkg/utils/env_utils.go`:

```go
if err := godotenv.Load(); err != nil {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("load .env file: %w", err)
}
```

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

[Unreleased]: https://github.com/samznd/layrgo/compare/v1.1.0...HEAD
[v1.1.0]: https://github.com/samznd/layrgo/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/samznd/layrgo/releases/tag/v1.0.0
