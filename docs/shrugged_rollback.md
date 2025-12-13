## shrugged rollback

Rollback the last applied migration(s)

### Synopsis

Rollback one or more migrations using their corresponding .down.sql files.

```
shrugged rollback [flags]
```

### Options

```
  -n, --count int   number of migrations to rollback (default 1)
      --dry-run     preview rollback without executing
  -h, --help        help for rollback
```

### Options inherited from parent commands

```
  -c, --config string             config file path (default "shrugged.yaml")
      --migrations-dir string     path to migrations directory
      --postgres-version string   postgres version for Docker containers
      --schema string             path to schema file
      --url string                database connection URL
```

### SEE ALSO

* [shrugged](shrugged.md)	 - PostgreSQL schema migration tool

