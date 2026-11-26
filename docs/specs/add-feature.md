# Spec: `layrgo add feature`

Status: implemented · Target: v1.5.0

## Goal

Add a complete feature to an existing Layrgo project with one command: model,
store, service, handler, routes and a test, wired into `cmd/main.go`, and
checked to build. It serves two users with one command:

- **Developers** get a guided, safe way to add features without copying
  `internal/message` by hand.
- **AI agents** get a predictable command to call, so they spend their effort on
  business rules instead of rewriting boilerplate and wiring.

Non-goals for v1.5.0: update and delete endpoints, relations between features,
pagination, auth, changing or removing an existing feature, an MCP server and
`layrgo check`. Each can follow later.

## Usage

```bash
layrgo add feature order --fields "customer:string,total:int,paid:bool"
```

Run from the project root (the directory with `go.mod`).

| Argument / flag | Meaning |
| --- | --- |
| `<name>` | Singular feature name, e.g. `order`. See [Names](#names). |
| `--fields` | Comma-separated `name:type` list. See [Fields](#fields). Required in non-interactive mode. |
| `--plural` | Override the derived plural, e.g. `--plural people` for `person`. |
| `--dry-run` | Print the files and the `main.go` diff; write nothing. |
| `--force` | Replace an existing `internal/<name>` package. Never touches other features. |
| `--json` | Print the result as JSON instead of text (for agents and scripts). |
| `--yes` | Never prompt; fail on anything missing instead. |

### Interactive mode

With no `--fields`, a terminal on stdin, and no `--yes`, the command asks:

```
? Feature name: order
? Fields (name:type, comma-separated): customer:string, total:int, paid:bool
```

It never prompts when stdin is not a terminal or `--yes` is set. A prompt an
agent cannot answer would hang it, which is the worst failure for an agent.

## Names

From `order` the command derives:

| Item | Value |
| --- | --- |
| Package and folder | `order` → `internal/order/` |
| Go type | `Order` |
| Route | `/api/v1/orders` and `/api/v1/orders/{id}` |
| Table | `orders` |
| Variables in `main.go` | `orderStore`, `orderService`, `orderHandler` |

The plural adds `s`, or `es` after `s`, `x`, `ch`, `sh`, or turns a final
consonant + `y` into `ies`. `--plural` overrides it.

The table is always the plural. That matters: `order` is a reserved word in SQL,
and XORM's default mapper would name the table `order`, so the XORM model sets
its table name explicitly.

A name is rejected (exit 1) unless it is lowercase letters and digits starting
with a letter, and is not a Go keyword, a predeclared identifier, or one of
`main`, `config`, `utils`, `middleware`, `internal`, `cmd`, `pkg`.

## Fields

`--fields "customer:string,note:string?,total:int,paid:bool"`

| Type | Go | Postgres | MySQL | SQLite |
| --- | --- | --- | --- | --- |
| `string` | `string` | `TEXT` | `VARCHAR(255)` | `TEXT` |
| `text` | `string` | `TEXT` | `TEXT` | `TEXT` |
| `int` | `int64` | `BIGINT` | `BIGINT` | `INTEGER` |
| `float` | `float64` | `DOUBLE PRECISION` | `DOUBLE` | `REAL` |
| `bool` | `bool` | `BOOLEAN` | `BOOLEAN` | `INTEGER` |
| `time` | `time.Time` | `TIMESTAMPTZ` | `DATETIME` | `TIMESTAMP` |

- Field names are `snake_case` in JSON and SQL, `PascalCase` in Go:
  `created_by` → `CreatedBy`, JSON `"created_by"`.
- Every feature gets an `id` (int64, auto-increment primary key) on its own; it
  cannot be declared.
- Fields are `NOT NULL`. A trailing `?` (`note:string?`) makes one optional: a
  pointer in Go and a nullable column.
- Validation the command generates: required `string` and `text` fields must not
  be blank after trimming. Everything else is left to the developer, at a
  `TODO` in the service.

## What it generates

All in `internal/<name>/`, following the conventions of `internal/message` and
`AGENTS.md`, for the project's framework, database and ORM:

| File | Contents |
| --- | --- |
| `model.go` | The type, `ErrNotFound`, `ErrInvalid` |
| `store.go` | `Migrate` (creates the table), `Create`, `List`, `Get` |
| `service.go` | The `store` interface it needs, validation, `TODO: business rules` |
| `service_test.go` | Fake store; tests for a valid create and each required field |
| `handler.go` | `Create` (201), `List` (200, `[]` when empty), `Get` (200/400/404) |
| `routes.go` | `SetupRoutes`, using that framework's prefix rules |

`List` runs one query; it never queries per row. `TODO` comments mark exactly
where custom logic goes, so an agent can search for them and a developer sees
them at once:

```go
func (s *Service) Create(ctx context.Context, o Order) (Order, error) {
	o.Customer = strings.TrimSpace(o.Customer)
	if o.Customer == "" {
		return Order{}, fmt.Errorf("%w: customer is required", ErrInvalid)
	}
	// TODO: business rules for creating an order.
	return s.store.Create(ctx, o)
}
```

## Wiring `cmd/main.go`

From v1.5.0, `layrgo init` writes three marker comments into `cmd/main.go`:

```go
	// layrgo:wire  (add feature inserts store, Migrate, service and handler above this line)
	...
	// layrgo:routes  (add feature inserts SetupRoutes calls above this line)
```

`add feature` inserts its lines above each marker and adds the import, then runs
`gofmt` on the file. It edits nothing else.

Projects created before v1.5.0 have no markers. For those, it inserts after the
last `<x>Handler := <x>.NewHandler(` line and the last `<x>.SetupRoutes(` line.
If it finds neither, or the file does not parse, it changes nothing, prints the
exact lines to paste, and exits 3.

## Knowing the project's stack

`add feature` needs the framework, database and ORM. From v1.5.0 `layrgo init`
records them in `.layrgo.json` at the project root:

```json
{"version": 1, "framework": "gin", "database": "postgres", "orm": "none"}
```

For older projects without the file, it reads `go.mod`:

- **Framework:** gin, echo, fiber, chi or iris from its module path; none of them means `stdlib`.
- **Database:** the driver (`lib/pq`, `go-sql-driver/mysql`, `go-sqlite3`, or a `gorm.io/driver/*`).
- **ORM:** `gorm.io/gorm`, `xorm.io/xorm` or `entgo.io/ent`, else `none`.

It then writes `.layrgo.json` so the next run does not need to guess. If the
stack cannot be determined, it exits 1 and says which part is missing.

## After writing

It runs, from the project root: `gofmt -l`, `go build ./...`, `go vet ./...`
and `go test ./internal/<name>/`, and reports each result.

## Output

Text (default):

```
✓ Created internal/order/ model.go store.go service.go service_test.go handler.go routes.go
✓ Wired order in cmd/main.go (store, Migrate, service, handler, routes)
✓ gofmt, go build, go vet and go test passed

Next: add your business rules in internal/order/service.go (search for TODO)
Try:  curl -X POST localhost:3000/api/v1/orders -d '{"customer":"Ada","total":100,"paid":false}'
```

`--json`:

```json
{
  "feature": "order",
  "files": ["internal/order/model.go", "internal/order/store.go", "..."],
  "wired": true,
  "routes": ["POST /api/v1/orders", "GET /api/v1/orders", "GET /api/v1/orders/{id}"],
  "checks": {"gofmt": true, "build": true, "vet": true, "test": true},
  "todos": ["internal/order/service.go:24"]
}
```

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Created, wired, all checks passed |
| 1 | Invalid input, or not a Layrgo project; nothing written |
| 2 | `internal/<name>` already exists (without `--force`); nothing written |
| 3 | Files created, but wiring was skipped (lines printed to paste) |
| 4 | Files created and wired, but a check failed (output included) |

## AGENTS.md changes

The "Adding a feature" section starts with the command:

> Run `layrgo add feature <name> --fields "…" --yes`, then fill in the `TODO`s.
> Don't copy `internal/message` by hand when the command is available.

The existing bookmark walkthrough stays below it, as the reference for what the
command generates and for changes it does not cover.

## Testing the implementation

- Unit tests for parsing names and fields, pluralization and rejected names.
- Golden tests of the generated files for each framework × ORM.
- For every framework × ORM on SQLite: generate a project, add a feature, then
  build, vet, test and call its endpoints live, as the v1.4.0 checks did. Also
  live on Postgres for each ORM, including a feature named `order`. (Done for
  all three databases; see [Verification](#verification).)
- Wiring: a v1.5.0 project (markers), a v1.4.0 project (no markers), and a
  `main.go` changed by hand so neither is found (expect exit 3 and no edit).
- An agent experiment like v1.4.0's, with and without the command, comparing
  tokens, structure and failures.

## Decisions

The open questions were settled as proposed:

1. The name is singular (`order`); a plural is derived, or set with `--plural`.
2. `.layrgo.json` is meant to be committed; `init` writes it and the command says
   to commit it when it creates one for an older project.
3. Required `string` and `text` fields get blank checks; other rules are `TODO`s.

## Verification

Run live for every framework × ORM (24 combinations) on SQLite, Postgres 14 and
MySQL 26.10, with a feature named `order` and one field of each type, required
and optional. Each run scaffolds a project, adds the feature, builds and starts
it, and calls every endpoint: an empty list returns `[]`, create returns 201, a
blank required field returns 400, list and get return 200, a missing id 404, a
non-numeric id 400, and the sample `/message` endpoint still returns 200.
Stored values, floats and times included, read back unchanged. The tables match
the types above, except where an ORM picks its own: GORM stores `float` as
`numeric` on Postgres and `time` as `datetime(3)` on MySQL, and XORM stores
`time` without a time zone on Postgres.

The Postgres run found that generated projects could not connect with an empty
`DB_PASSWORD`. The fix, which builds the connection string as an escaped URL,
ships after v1.5.0; see the changelog.

Not covered yet: golden files per stack (tests check each stack's key lines
instead), and the agent experiment.
