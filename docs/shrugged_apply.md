## shrugged apply

Apply pending migrations to the database

### Synopsis

Apply all pending migrations to the database in order. Use --dry-run to preview without applying.

```
shrugged apply [flags]
```

### Options

```
      --dry-run   preview migrations without applying
      --force     apply even if previous migrations have been modified
  -h, --help      help for apply
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

