# Build from source
1. Clone the repo
2. Run `go build -o ./.out/monotrack ./main.go`

# Usage
1. Run `monotrack init` to create a template configuration (monotrack.yaml, the .monotrack-manifest.yaml is a WIP)
2. Edit the config file to match your actual paths
3. Run `monotrack <baseSHA> <HEAD>` to list packages that changed

> **_NOTE:_**  you will see other commands available, these are not yet implemented.

## Example
Given the following monotrack.yaml:
```
projects:
  frontend:
    type: node
    path: apps/frontend
  backend:
    type: go
    path: apps/backend
    dependsOn:
      - shared-package
  shared-package:
    type: go
    path: packages/shared
    dependsOn:
      - another-shared
  another-shared:
    type: go
    path: packages/another-shared
```

An update to a file in the `packages/another-shared` package, will result in the following output:
```
$ monotrack c4688b6a4aa2d3a50a0e1ec59c69d0eeacee36b6 428cb452e22252ebee05e6ee8209175f330b16aa
another-shared
shared-package
backend
```

# TODO
- [ ] dynamically generate monotrack.yaml
- [ ] keep track of versions/tags in the .monotrack-manifest.yaml
- [ ] implement other helper commands
- [ ] different output formats for root command (by name, by path, by tag, etc)
