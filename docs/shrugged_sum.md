## shrugged sum

Regenerate the sum file from migration files

### Synopsis

Regenerate shrugged.sum by hashing all migration files on disk.

Use this after resolving merge conflicts that leave the sum file
out of sync with the actual migration files.

```
shrugged sum [flags]
```

### Options

```
  -h, --help   help for sum
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

