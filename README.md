🤷‍♂️ shrugged is a declarative PostgreSQL schema migration tool that works fully offline.

## Why shrugged?

- Declarative schema → forward and down migration generation
- Schema diffing for all PostgreSQL objects
- Migration apply/status tracking
- Drift detection and validation
- Fully offline and authentication-free

## Installation

```bash
curl -sSL shrugged.terminallyonline.io/install.sh | sh
```

Or with Go:

```bash
go install github.com/terminally-online/shrugged/cmd/shrugged@latest
```

<details>
<summary>Build from source</summary>

```bash
git clone https://github.com/terminally-online/shrugged
cd shrugged
go build -o shrugged ./cmd/shrugged
```
</details>

**Requirements:**
- Go 1.21+
- Docker (for diffing and migration generation)
- PostgreSQL 14+ (target database)

## Quick Start

1. Create a `shrugged.yaml`:

```yaml
database_url: ${DATABASE_URL}
schema: schema.sql
migrations_dir: migrations
postgres_version: "16"
```

2. Define your schema in `schema.sql`:

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);
```

3. Generate a migration:

```bash
shrugged migrate
```

4. Apply it:

```bash
shrugged apply
```
