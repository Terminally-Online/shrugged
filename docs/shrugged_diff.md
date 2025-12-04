## shrugged diff

Show differences between schema file and migrations

### Synopsis

Compare the declarative schema file against the result of applying all migrations.

This spins up a temporary Postgres container, applies all migrations to get the
"current" state, then compares against the desired schema file.

```
shrugged diff [flags]
```

### Options

```
  -h, --help   help for diff
```

### Options inherited from parent commands

```
  -c, --config string   config file path (default "shrugged.yaml")
```

### SEE ALSO

* [shrugged](shrugged.md)	 - PostgreSQL schema migration tool

