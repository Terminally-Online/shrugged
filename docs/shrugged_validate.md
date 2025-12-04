## shrugged validate

Validate the schema file

### Synopsis

Validate the schema file by applying it to a temporary Postgres container.

This ensures the SQL is syntactically correct and can be executed against
the configured Postgres version.

```
shrugged validate [flags]
```

### Options

```
  -h, --help   help for validate
```

### Options inherited from parent commands

```
  -c, --config string   config file path (default "shrugged.yaml")
```

### SEE ALSO

* [shrugged](shrugged.md)	 - PostgreSQL schema migration tool

