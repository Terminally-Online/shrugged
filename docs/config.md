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
