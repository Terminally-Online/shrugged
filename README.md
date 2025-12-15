# shrugged

Declarative PostgreSQL migrations and type-safe Go code generation. Fully offline, no cloud dependencies.

## Features

- **Declarative schema** - Write your schema once, get forward and rollback migrations generated automatically
- **Schema diffing** - Supports all PostgreSQL objects (tables, indexes, functions, triggers, policies, etc.)
- **Code generation** - Generate type-safe Go models and query bindings from your database
- **Schema linting** - Catch issues like missing foreign key indexes before they hit production
- **Drift detection** - Know when your database has diverged from your migrations

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/terminally-online/shrugged/main/install.sh | sh
```

Or with Go:

```bash
go install github.com/terminally-online/shrugged/cmd/shrugged@latest
```

**Requirements:** Docker, PostgreSQL 14+

Once installed just run the command `shrugged` to see all the available options. 

## Documentation

- [Configuration](docs/config.md)
- [Commands](docs/shrugged.md)

## License

MIT
