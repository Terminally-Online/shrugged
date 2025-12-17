## shrugged generate

Generate language bindings from database schema

### Synopsis

Generate native language bindings (models/types) from the database schema.

The generator introspects the database and creates type-safe models for tables,
enums, and composite types in the specified language.

If no database URL is provided, a temporary Postgres container is started and
the schema file is applied automatically.

Use --clean to remove orphaned query files that no longer have corresponding SQL queries.

Example:
  shrugged generate --language go --out ./models
  shrugged generate --url postgres://localhost/mydb --language go --out ./models
  shrugged generate --clean

```
shrugged generate [flags]
```

### Options

```
      --clean                remove orphaned query files that no longer have corresponding SQL queries
  -h, --help                 help for generate
      --language string      target language (default: go)
      --out string           output directory for generated files
      --queries string       path to queries file or directory
      --queries-out string   output directory for query bindings
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

