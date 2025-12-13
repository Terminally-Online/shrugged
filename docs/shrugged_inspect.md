## shrugged inspect

Dump the current database schema

### Synopsis

Inspect the live database and output the current schema as SQL.

```
shrugged inspect [flags]
```

### Options

```
  -h, --help            help for inspect
  -o, --output string   output file (default: stdout)
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

