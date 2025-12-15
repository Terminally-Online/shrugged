## shrugged

PostgreSQL schema migration tool

### Synopsis

Shrugged is a PostgreSQL schema migration tool that provides
automatic schema diffing and migration generation.

No cloud dependencies. No paywalled features. Just migrations.

### Options

```
  -c, --config string             config file path (default "shrugged.yaml")
  -h, --help                      help for shrugged
      --migrations-dir string     path to migrations directory
      --postgres-version string   postgres version for Docker containers
      --schema string             path to schema file
      --url string                database connection URL
```

### SEE ALSO

* [shrugged apply](shrugged_apply.md)	 - Apply pending migrations to the database
* [shrugged diff](shrugged_diff.md)	 - Show differences between schema file and migrations
* [shrugged generate](shrugged_generate.md)	 - Generate language bindings from database schema
* [shrugged inspect](shrugged_inspect.md)	 - Dump the current database schema
* [shrugged migrate](shrugged_migrate.md)	 - Generate a migration from schema differences
* [shrugged rollback](shrugged_rollback.md)	 - Rollback the last applied migration(s)
* [shrugged status](shrugged_status.md)	 - Show migration status
* [shrugged validate](shrugged_validate.md)	 - Validate the schema file
* [shrugged version](shrugged_version.md)	 - Print the version number

