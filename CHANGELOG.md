# Changelog

All notable changes to this project are documented here.
This project follows [Semantic Versioning](https://semver.org/) (`vMAJOR.MINOR.PATCH`).

## v1.6.0 — 2026-11-26

### Added
- **`layrgo update agents`** brings the instructions for AI coding agents in an
  existing project up to date: it rewrites `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`
  and `.github/copilot-instructions.md` for the project's stack, so projects
  made by an older Layrgo learn about `layrgo add feature`. It backs up each file
  it changes (`AGENTS.md.bak`, ...) and never overwrites an earlier backup.
  `--dry-run` shows the diff; `--json` prints the result. For projects without
  `.layrgo.json`, it detects the stack from `go.mod` and records it.

### Changed
- `AGENTS.md` tells agents what to do when `layrgo` is not installed: install it
  with `go install`, or follow the hand-written walkthrough when they can't.
  It also points to `layrgo update agents` for refreshing the file.
- `--dry-run` diffs show a changed line as the old line followed by the new
  one, as `diff` does.

## v1.5.1 — 2026-11-26

### Fixed
- **Generated projects could not connect to Postgres with an empty
  `DB_PASSWORD`**, as is common for a local Postgres. `config/database.go` built
  a `key=value` connection string, so the driver read the next setting
  (`dbname=…`) as the password, and XORM rejected the string outright. It now
  builds a `postgres://` URL with `url.UserPassword`, which also escapes
  passwords containing spaces, quotes, `@` or `/`. MySQL and SQLite are
  unchanged.

## v1.5.0 — 2026-11-26

### Added
- **`layrgo add feature`** adds a complete feature to an existing project in one
  command: `layrgo add feature order --fields "customer:string,total:int,paid:bool"`
  writes `internal/order/` (model, store with `Migrate`, service with validation
  and `TODO`s, a service test, handler and routes) for the project's framework,
  database and ORM, wires it into `cmd/main.go`, and runs gofmt, go build, go vet
  and the feature's tests. It serves `POST`, `GET` (list) and `GET /{id}`. Flags:
  `--plural`, `--dry-run`, `--force`, `--json` and `--yes`; it never prompts when
  stdin is not a terminal. Exit codes 0–4 tell agents what happened.
- **`.layrgo.json`** records the project's stack. `layrgo init` writes it; for
  older projects, `add feature` detects the stack from `go.mod` and writes it.

### Changed
- `cmd/main.go` has `// layrgo:wire` and `// layrgo:routes` markers, where
  `add feature` inserts new features. Older projects without them are wired
  after their last feature.
- `AGENTS.md` tells agents to use `layrgo add feature`, and keeps the bookmark
  walkthrough as the reference for what it generates.

## v1.4.0 — 2026-11-18

### Added
- **Generated projects carry instructions for AI coding agents.** `AGENTS.md` is
  written for the project's exact framework, database and ORM: the commands that
  prove a change works, the layering rules, and a complete, compilable example of
  adding a feature with that stack's routing, SQL placeholders and table setup.
  Codex, Cursor, Windsurf, Zed, Amp and others read it directly. `CLAUDE.md` and
  `GEMINI.md` import it for Claude Code and Gemini CLI, and
  `.github/copilot-instructions.md` points GitHub Copilot to it.
- **The sample feature ships with a test.** `internal/message/service_test.go`
  tests the service against a fake store. Agents copy the sample feature's shape;
  in our runs without `AGENTS.md`, agents went from writing tests for a new
  feature 0 times in 5 to 5 times in 5 once the sample had one.

### Changed
- **The sample store creates its own table.** `internal/message/store.go` now has
  a `Migrate` method that creates and seeds a `messages` table (`message` with
  XORM), and `cmd/main.go` calls it at startup before wiring the feature. It is
  safe to run on every start. The response is unchanged:
  `{"message":"data from the store"}`.
- The XORM store's field is named `db`, like the other stores, instead of `engine`.
- Wiring variables in `cmd/main.go` are prefixed with their feature
  (`messageStore`, `messageService`, `messageHandler`), so a second feature can
  be wired alongside without a name clash.

### Fixed
- **Generated Go was not always gofmt-clean.** In `cmd/main.go`, a project name
  that sorts before `github.com` (for example `api`) left the imports unsorted;
  the project's own imports are now a separate group. `pkg/utils/env_utils.go`
  was missing its final newline.

## v1.3.1 — 2026-11-12

### Fixed
- **A test still expected the pre-v1.3.0 sample response.**
  `TestScaffold_RunsWithoutDotEnv` looked for `"data from repository"`, but v1.3.0
  changed the generated app's response to `"data from the store"`, so the test
  suite failed at the v1.3.0 tag. The generated code was correct; only the
  assertion was stale. No change to the CLI or to generated projects: v1.3.0 and
  v1.3.1 produce identical output.

## v1.3.0 — 2026-11-12

### Changed
- **Generated projects are packaged by feature instead of by layer.** The sample
  endpoint now lives in one package, `internal/message`, holding `handler.go`,
  `service.go`, `store.go` and `routes.go`. The previous `internal/handlers`,
  `internal/services`, `internal/repositories`, `internal/routes` and the empty
  `internal/models` are no longer generated. Projects created by earlier versions
  are unaffected; only the output of `layrgo init` changes.
- **The repository is now a `Store`.** `NewRepository` became `NewStore`, in
  `store.go`. The sample response changed with it, from
  `{"message":"data from repository"}` to `{"message":"data from the store"}`.
- **Interfaces are declared where they are used, and constructors return concrete
  types.** Previously `Repository` and `Service` were interfaces defined beside
  their only implementation, and their constructors returned the interface. Now
  the service declares the one-method `store` interface it needs, the handler
  declares what it needs from the service, and `NewStore`/`NewService` return
  `*Store`/`*Service`.

### Added
- Tests render the feature package for every framework and ORM and require the
  output to be gofmt-clean Go.

## v1.2.1 — 2026-11-09

### Fixed
- **`--framework` help text now lists every supported framework.** v1.2.0 added
  `stdlib` to the validation list and the interactive wizard but not to the flag's
  own description, so `layrgo init --help` still read "Fiber, Gin, Echo, Chi, or
  Iris" and the new framework was undiscoverable from the CLI. The feature itself
  worked. All three now read one `cli.Frameworks` slice, and tests assert the help
  text lists every entry and that every entry has renderable templates behind it.

## v1.2.0 — 2026-11-09

### Added
- **`net/http` as a sixth framework, selected with `--framework stdlib`.** Generated
  projects route with `http.ServeMux` method patterns (`"GET /api/v1/message"`), which
  Go 1.22 added, so the stack pulls in **no router dependency at all** — a SQLite +
  `database/sql` project now needs only `godotenv` and the driver. The generated
  `main.go` uses an explicit `http.Server` with read, write, header and idle timeouts
  rather than `http.ListenAndServe`, which sets none.
- Scaffolding with `--framework stdlib` on a toolchain older than Go 1.22 is now
  refused up front, before any files are written. Pre-1.22 `ServeMux` treats
  `"GET /path"` as a literal path, so the project would compile and run while every
  request 404s; the error names the required version and the alternatives.

## v1.1.1 — 2026-11-09

### Fixed
- **`layrgo version` reports the real version when installed with `go install`.**
  The version, commit and date were supplied exclusively by `-ldflags -X`, which only
  the `Makefile` and GoReleaser pass, so a binary from
  `go install github.com/samznd/layrgo/cmd/layrgo@v1.1.0` — the install method the
  README documents — printed `layrgo dev (commit: none, built: unknown)`. The values
  now fall back to `runtime/debug.ReadBuildInfo()`: the module version for a
  `go install <module>@<version>` build, and the `vcs.revision` / `vcs.time` build
  settings for a build from a working tree. Linker-provided values still win when
  present, so `make build` and release archives are unaffected.

### Changed
- Version output is normalized to the `v`-prefixed form. GoReleaser stamps
  `{{ .Version }}`, which has no leading `v`, so release binaries reported `layrgo 1.1.0`
  while module versions and git tags are written `v1.1.0`; both paths now agree.
- The `(commit: …, built: …)` detail is omitted when neither value is known, instead of
  printing the `none` / `unknown` placeholders. A `go install <module>@<version>` build
  resolves a real version but records no VCS metadata, because a module zip contains no
  repository.
- A build from a modified working tree marks its commit `-dirty`, matching what
  `git describe --dirty` already reported on the `Makefile` path.

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
