# Layrgo

[![Go Reference](https://pkg.go.dev/badge/github.com/samznd/layrgo.svg)](https://pkg.go.dev/github.com/samznd/layrgo)
[![GitHub release](https://img.shields.io/github/v/release/samznd/layrgo.svg)](https://github.com/samznd/layrgo/releases)
[![CI](https://github.com/samznd/layrgo/actions/workflows/ci.yml/badge.svg)](https://github.com/samznd/layrgo/actions/workflows/ci.yml)

<p align="center">
  <img src="docs/assets/layrgo-banner.png" width="600" alt="Layrgo: one command scaffolds a layered Go API, then an AI agent adds an orders feature in the same shape, with a passing test">
</p>

**Scaffold a Go API. Let AI build on it.**

Layrgo is a CLI that generates a complete, compilable Go web project — handler, service and data layers already wired together, dependency injection, `.env` config, Docker, and pinned dependencies — so you start building features instead of wiring boilerplate.

Every project also ships with instructions for AI coding agents, so Claude Code, Cursor, Codex, Copilot and Gemini CLI add features in the same structure, with tests.

## Why Layrgo?

| Without Layrgo                          | With Layrgo                                  |
| --------------------------------------- | -------------------------------------------- |
| Manually wire handlers, services, stores | Pre-wired `handler → service → store`       |
| Pick framework + DB + ORM from scratch  | Choose your stack interactively or via flags |
| Write Dockerfile and compose yourself   | Docker + compose generated per database      |
| Hope the layout is idiomatic            | Standard Go project layout out of the box    |
| AI agents invent a new structure each time | `AGENTS.md` keeps them in your project's shape |

**Supported stacks:** Fiber · Gin · Echo · Chi · Iris · `net/http` × Postgres · MySQL · SQLite × GORM · XORM · Ent · raw `database/sql`

## Quick start

```bash
go install github.com/samznd/layrgo/cmd/layrgo@latest
layrgo init
```

### Non-interactive (scripts, demos, CI)

```bash
layrgo init myapi \
  --framework gin \
  --database Postgres \
  --orm GORM

cd myapi
go run ./cmd
curl http://localhost:3000/api/v1/message
# {"message":"data from the store"}
```

Then open the project in your AI coding tool and ask it to *"add an orders feature"*.
It reads `AGENTS.md` and builds the feature in the project's own shape, with a test.
See [Works with AI coding agents](#works-with-ai-coding-agents).

### Dependency-free routing (`net/http`)

`--framework stdlib` generates a project that routes on the standard library, using
the `http.ServeMux` method patterns added in Go 1.22 — so no router is pulled in:

```bash
layrgo init myapi --framework stdlib --database SQLite --no-orm

cd myapi
cat go.mod        # godotenv + the SQLite driver. No router.
go run ./cmd

curl      http://localhost:3000/api/v1/message   # 200 {"message":"data from the store"}
curl -X POST http://localhost:3000/api/v1/message # 405: the GET pattern routes by method
```

The generated `main.go` uses an explicit `http.Server` with read, write, header and
idle timeouts, rather than `http.ListenAndServe`, which sets none.

### Interactive example

```bash
$ layrgo init
? What is your project name? mywebapp
? Choose your web framework: Gin
? Choose your database system: Postgres
? Would you like to use an ORM? Yes
? Choose your ORM framework: GORM

✅ Project "mywebapp" is ready (Gin + Postgres + GORM)

Next steps:
  cd mywebapp
  go run ./cmd
  curl http://localhost:3000/api/v1/message
```

## Works with AI coding agents

Every generated project includes an `AGENTS.md` written for its exact stack. It
tells an agent how the project is laid out, the rules that keep the layers
apart, the commands that prove a change works, and a complete, compilable
example of adding a feature, using that framework's routing, that database's
SQL placeholders and that ORM's migrations.

One file serves every tool. Codex, Cursor, Windsurf, Zed, Amp and others read
`AGENTS.md` directly; `CLAUDE.md` and `GEMINI.md` import it for Claude Code and
Gemini CLI, and `.github/copilot-instructions.md` points GitHub Copilot to it.
Edit `AGENTS.md` and every tool follows the change.

The sample feature teaches by example too: it ships with a test
(`service_test.go`) and creates its own table at startup (`Migrate`), and agents
copy what they see.

### What we measured

We gave AI agents (Claude Sonnet) the same task, "add an orders feature backed by
Postgres", five times per setup, and tested every result against a real Postgres:

| | Plain Go module | Layrgo + AGENTS.md |
|---|---|---|
| Works end to end | 5/5 | 5/5 |
| Follows `handler → service → store` | 0/5 | 5/5 |
| Wrote tests for the feature | 4/5 | 5/5 |
| Same file layout every run | no | yes |

Before the sample feature shipped with a test, agents working in a Layrgo
project without `AGENTS.md` wrote tests 0 times in 5; with it, 5 times in 5. The
cost of `AGENTS.md` is roughly 7k extra tokens per feature. These are small
samples from one model and one stack, so treat them as a signal, not a benchmark.

## Installation

**Requirements:** Go 1.23+ to install the CLI. Generated projects target your local
Go version; the `stdlib` framework needs Go 1.22+ for `net/http` routing (layrgo
refuses the combination on older toolchains rather than generating a project whose
routes silently 404).

```bash
go install github.com/samznd/layrgo/cmd/layrgo@latest
```

Or build from source:

```bash
git clone https://github.com/samznd/layrgo.git
cd layrgo
make install
layrgo version
```

## Generated project layout

Layrgo packages code **by feature**: everything about one feature lives
in one package, so a change to it touches one directory.

```
your-project/
├── cmd/main.go              # Entry point with DI wiring
├── config/database.go       # DB connection for your stack
├── internal/
│   ├── message/             # One package per feature
│   │   ├── handler.go       # HTTP layer
│   │   ├── service.go       # Business logic
│   │   ├── service_test.go  # Service test with a fake store
│   │   ├── store.go         # Data access, plus Migrate for its table
│   │   └── routes.go        # Route registration
│   └── middleware/          # Ready for your middleware
├── pkg/utils/               # Env loader
├── Dockerfile
├── docker-compose.yml
├── .gitignore
├── .env.example
├── .env                     # gitignored; local overrides
├── .layrgo.json             # The stack, read by `layrgo add feature`; commit it
├── AGENTS.md                # Instructions for AI coding agents
├── CLAUDE.md                # Claude Code: imports AGENTS.md
├── GEMINI.md                # Gemini CLI: imports AGENTS.md
└── .github/copilot-instructions.md  # GitHub Copilot: points to AGENTS.md
```

To add a feature, run `layrgo add feature` (below). It writes `internal/<name>`
for your stack, wires it into `cmd/main.go` above the `// layrgo:wire` and
`// layrgo:routes` markers, and checks that the project still builds and passes
its tests. `AGENTS.md` walks through the same code by hand, for changes the
command does not cover.

Interfaces are declared by the code that uses them, and constructors return
concrete types ("accept interfaces, return structs"). The service declares the
one-method `store` interface it needs; `NewStore` returns a `*Store`. A test can
hand the service a fake store without a database, as the generated
`service_test.go` does.

## Commands

```bash
layrgo init                  # interactive wizard
layrgo init myapp --flags    # non-interactive scaffold
layrgo add feature <name> --fields "..."   # add a feature to the project in .
layrgo update agents         # refresh AGENTS.md and friends after upgrading
layrgo version               # print version
layrgo --help
```

### `init` flags

| Flag          | Description                    |
| ------------- | ------------------------------ |
| `--framework` | Fiber, Gin, Echo, Chi, Iris, or `stdlib` (net/http, no router dependency; needs Go 1.22+) |
| `--database`  | Postgres, MySQL, or SQLite     |
| `--orm`       | GORM, XORM, or Ent             |
| `--no-orm`    | Use `database/sql` instead     |
| `--force`     | Scaffold into a non-empty directory |

### `add feature`

Run from the project root (the directory with `go.mod`):

```bash
layrgo add feature order --fields "customer:string,note:text?,total:int,paid:bool"
```

This creates `internal/order/` with `model.go`, `store.go`, `service.go`,
`service_test.go`, `handler.go` and `routes.go`, serving `POST /api/v1/orders`,
`GET /api/v1/orders` and `GET /api/v1/orders/{id}` from an `orders` table.

| Flag | Description |
| --- | --- |
| `--fields` | `name:type` list. Types: `string`, `text`, `int`, `float`, `bool`, `time`. A trailing `?` makes a field optional (nullable). Every feature gets an `id`. |
| `--plural` | Override the derived plural, e.g. `--plural people` for `person` |
| `--dry-run` | Print the files and the `cmd/main.go` diff; write nothing |
| `--force` | Replace an existing `internal/<name>` package |
| `--json` | Print the result as JSON (for agents and scripts) |
| `--yes` | Never prompt; fail on anything missing instead |

Without `--fields`, it asks for them when run at a terminal. It never prompts
when stdin is not a terminal, so agents and CI can't hang on it.

Exit codes: `0` created, wired and checked · `1` invalid input or not a Layrgo
project, nothing written · `2` the feature exists (without `--force`) · `3`
files created but `cmd/main.go` could not be wired (the lines to paste are
printed) · `4` created and wired, but gofmt, build, vet or test failed.

Projects created before v1.5.0 work too: the stack is read from `go.mod` and
saved to `.layrgo.json`, and the feature is wired after the last existing one.

### `update agents`

Projects created by an older Layrgo keep the agent instructions of that
version, so their agents don't know about newer commands such as
`add feature`. After upgrading, run from the project root:

```bash
layrgo update agents --dry-run   # show what would change
layrgo update agents
```

It rewrites `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` and
`.github/copilot-instructions.md` for the project's stack. Each file it
changes is backed up first (`AGENTS.md.bak`, then `AGENTS.md.bak.2`, ...), so
copy anything you added by hand from the backup, then delete it. Files already
up to date are left alone. `--json` prints the result for scripts.

## Layrgo repository layout

```
layrgo/
├── cmd/layrgo/          # CLI entry point
├── internal/
│   ├── cli/             # Commands (init, add feature, update agents, version)
│   ├── scaffold/        # Scaffolding engine
│   ├── feature/         # add feature: names, fields, wiring, checks
│   ├── project/         # An existing project's module and stack (.layrgo.json)
│   ├── agents/          # update agents: rewrites the agent instruction files
│   ├── textdiff/        # Unified diffs for --dry-run
│   ├── templates/       # Template rendering
│   │   └── files/       # The templates, AGENTS.md's included
│   └── fileutil/
├── docs/assets/         # README images
├── scripts/             # Git hooks and their installer
└── .github/workflows/   # CI + release
```

## Development

```bash
make build          # build to bin/layrgo
make run            # build, then run bin/layrgo
make test           # unit tests
make test-integration  # scaffold + go build smoke test
make vet
make clean          # remove bin/
```

Run `scripts/install-hooks.sh` once after cloning to install the repo's git
hooks. Its `commit-msg` hook removes Cursor's co-author trailer from commit
messages.

## Releases

Versioning follows [Semantic Versioning](https://semver.org/) with `v`-prefixed git tags (`v1.0.0`, `v1.1.0`, …, `v1.6.0`).

```bash
# tag a release (triggers GoReleaser via GitHub Actions)
git tag -a v1.6.0 -m "v1.6.0"
git push origin v1.6.0
```

See [CHANGELOG.md](CHANGELOG.md) for release notes.

## Roadmap

- [x] Multi-framework support (Fiber, Gin, Echo, Chi, Iris, net/http)
- [x] Docker + compose per database
- [x] Compilable generated output with DI
- [x] Non-interactive `init` flags
- [x] AGENTS.md for AI coding agents (Claude Code, Cursor, Codex, Copilot, Gemini)
- [x] Sample feature with a test and startup table setup, for agents to copy
- [x] `layrgo add feature` for agents and humans
- [x] `layrgo update agents`, so older projects get the current agent instructions
- [ ] Sample store test, so agents test their SQL too
- [ ] MCP server, so agents can call Layrgo as a tool
- [ ] GraphQL scaffold
- [ ] gRPC scaffold
- [ ] SQLBoiler support

## License

MIT — see [LICENSE](LICENSE).

## Acknowledgments

- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Cobra](https://github.com/spf13/cobra) · [Survey](https://github.com/AlecAivazis/survey)
