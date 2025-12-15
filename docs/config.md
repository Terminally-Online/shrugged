## Configuration

All options can be set via config file or command-line flags. Flags take precedence over config values.

### Config File

By default, shrugged looks for `shrugged.yaml` in the current directory. Use `-c` or `--config` to specify a different path.

```yaml
database_url: ${DATABASE_URL}
schema: schema.sql
migrations_dir: migrations
postgres_version: "16"
```

All fields are optional. Environment variables are expanded using `${VAR}` or `$VAR` syntax.

### Command-Line Flags

Global flags available on all commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--url` | Database connection URL | - |
| `--schema` | Path to schema file | `schema.sql` |
| `--migrations-dir` | Path to migrations directory | `migrations` |
| `--postgres-version` | Postgres version for Docker containers | `16` |
| `-c, --config` | Config file path | `shrugged.yaml` |

### Precedence

1. Command-line flags (highest)
2. Config file values
3. Default values (lowest)

### Examples

Using config file:
```bash
shrugged migrate
shrugged apply
```

Using flags only (no config file needed):
```bash
shrugged migrate --schema internal/sql/tables.sql --postgres-version 15
shrugged apply --url postgres://localhost/mydb
```

Overriding config with flags:
```bash
shrugged apply --url postgres://localhost/staging
```

### Which Commands Need What

| Command | Requires `--url` | Uses Docker |
|---------|------------------|-------------|
| `validate` | No | Yes |
| `migrate` | No | Yes |
| `diff` | No | Yes |
| `apply` | Yes | No |
| `status` | Yes | No |
| `rollback` | Yes | No |
| `inspect` | Yes | No |
| `generate` | Yes | No |

### Generate Command

The `generate` command creates Go models and query bindings from your database schema.

#### Generate Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--out` | Output directory for generated models | `models` |
| `--language` | Target language (currently `go`) | `go` |
| `--queries` | Path to SQL file with query definitions | - |
| `--queries-out` | Output directory for query bindings | `queries` |

#### Examples

Generate models only:
```bash
shrugged generate --url postgres://localhost/mydb --out ./db/models
```

Generate models and query bindings:
```bash
shrugged generate \
  --url postgres://localhost/mydb \
  --out ./db/models \
  --queries ./db/queries.sql \
  --queries-out ./db/queries
```

#### Query File Format

Queries are defined with annotations:

```sql
-- name: QueryName :resulttype
-- Optional: nest: StructName(prefix.*)

SELECT ...
```

Result types:
- `:row` - Returns single row (`*ResultRow, error`)
- `:rows` - Returns multiple rows (`[]ResultRow, error`)
- `:exec` - No result, just execute (`error`)
- `:execrows` - Returns affected row count (`int64, error`)

Named parameters use `@param` syntax:
```sql
-- name: GetUserByEmail :row
SELECT * FROM users WHERE email = @email;
```

Optional filter pattern (parameter becomes a pointer):
```sql
-- name: ListUsers :rows
SELECT * FROM users
WHERE (status = @status OR @status IS NULL);
```
