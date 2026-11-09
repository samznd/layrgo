# Layrgo

[![Go Reference](https://pkg.go.dev/badge/github.com/samznd/layrgo.svg)](https://pkg.go.dev/github.com/samznd/layrgo)
[![GitHub release](https://img.shields.io/github/v/release/samznd/layrgo.svg)](https://github.com/samznd/layrgo/releases)
[![CI](https://github.com/samznd/layrgo/actions/workflows/ci.yml/badge.svg)](https://github.com/samznd/layrgo/actions/workflows/ci.yml)

**Scaffold production-ready Go web APIs in seconds.**

Layrgo is a CLI that generates a complete, compilable Go web project — clean architecture layers, dependency injection, `.env` config, Docker, and pinned dependencies — so you start building features instead of wiring boilerplate.

## Why Layrgo?

| Without Layrgo                          | With Layrgo                                  |
| --------------------------------------- | -------------------------------------------- |
| Manually wire handlers, services, repos | Pre-wired `handler → service → repository`   |
| Pick framework + DB + ORM from scratch  | Choose your stack interactively or via flags |
| Write Dockerfile and compose yourself   | Docker + compose generated per database      |
| Hope the layout is idiomatic            | Standard Go project layout out of the box    |

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
# {"message":"data from repository"}
```

### Dependency-free routing (`net/http`)

`--framework stdlib` generates a project that routes on the standard library, using
the `http.ServeMux` method patterns added in Go 1.22 — so no router is pulled in:

```bash
layrgo init myapi --framework stdlib --database SQLite --no-orm

cd myapi
cat go.mod        # godotenv + the SQLite driver. No router.
go run ./cmd

curl      http://localhost:3000/api/v1/message   # 200 {"message":"data from repository"}
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

```
your-project/
├── cmd/main.go              # Entry point with DI wiring
├── config/database.go       # DB connection for your stack
├── internal/
│   ├── handlers/            # HTTP layer
│   ├── services/            # Business logic
│   ├── repositories/        # Data access
│   ├── routes/              # Route registration
│   ├── middleware/          # Ready for your middleware
│   └── models/              # Ready for your models
├── pkg/utils/               # Env loader
├── Dockerfile
├── docker-compose.yml
├── .gitignore
├── .env.example
└── .env                     # gitignored; local overrides
```

## Commands

```bash
layrgo init                  # interactive wizard
layrgo init myapp --flags    # non-interactive scaffold
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

## Layrgo repository layout

```
layrgo/
├── cmd/layrgo/          # CLI entry point
├── internal/
│   ├── cli/             # Commands (init, version)
│   ├── scaffold/        # Scaffolding engine
│   ├── templates/       # Generated app templates
│   └── fileutil/
└── .github/workflows/   # CI + release
```

## Development

```bash
make build          # build to bin/layrgo
make test           # unit tests
make test-integration  # scaffold + go build smoke test
make vet
```

## Releases

Versioning follows [Semantic Versioning](https://semver.org/) with `v`-prefixed git tags (`v1.0.0`, `v1.1.0`, `v1.2.0`, …).

```bash
# tag a release (triggers GoReleaser via GitHub Actions)
git tag -a v1.2.1 -m "v1.2.1"
git push origin v1.2.1
```

See [CHANGELOG.md](CHANGELOG.md) for release notes.

## Roadmap

- [x] Multi-framework support (Fiber, Gin, Echo, Chi, Iris, net/http)
- [x] Docker + compose per database
- [x] Compilable generated output with DI
- [x] Non-interactive `init` flags
- [ ] GraphQL scaffold
- [ ] gRPC scaffold
- [ ] SQLBoiler support

## License

MIT — see [LICENSE](LICENSE).

## Acknowledgments

- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Cobra](https://github.com/spf13/cobra) · [Survey](https://github.com/AlecAivazis/survey)
