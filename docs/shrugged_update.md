## shrugged update

Update shrugged to the latest release

### Synopsis

Download and install the latest shrugged release from GitHub.

The binary is verified against its SHA256 checksum before installation.
If shrugged is installed in a system directory, re-run with sudo.

```
shrugged update [flags]
```

### Options

```
  -h, --help   help for update
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

