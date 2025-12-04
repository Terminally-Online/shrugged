## shrugged migrate

Generate a migration from schema differences

### Synopsis

Compare the schema file to the migrations and generate a new migration file.

This spins up a temporary Postgres container, applies all existing migrations,
then diffs against the desired schema to produce a new migration.

```
shrugged migrate [flags]
```

### Options

```
  -h, --help   help for migrate
```

### Options inherited from parent commands

```
  -c, --config string   config file path (default "shrugged.yaml")
```

### SEE ALSO

* [shrugged](shrugged.md)	 - PostgreSQL schema migration tool

