## Purpose
While there already exist plenty of tools to help with code verisoning, support across languages, specifically for monorepos, may be considered insufficient.  
I built Monotrack because I needed a generic and simple solution to help with versioning in a monorepo. Especially for testing purposes.  
This project is currently in it's testing phase and has some limitations. For example having to manually specify a local/internal dependency tree.  
Though this is a limitation other versioning tools also observe.  
A solution I might implement is automatic generation of dependency trees.  
However this would of course require manually implementing the automation for each lanugage individually.  
For now, my main use cases are easily creating pre-release/development tags for development branches,
and only running actions for projects that actually changed, without having to rely on a heavier tool like release-please or release-it.  

## Installation

### Build from source
1. Clone the repository.
2. Run:
   ```bash
   go build -o ./monotrack ./main.go
   ```

### Download binary
```bash
curl -LO https://github.com/ArnoldVanN/monotrack/releases/download/v0.2.0/monotrack_Linux_x86_64.tar.gz
tar -xzf monotrack_Linux_x86_64.tar.gz
mv monotrack /usr/local/bin/
```

# Usage
> **_Warn:_** Currently, Monotrack requires pre-existing git tags to be available for certain commands and in the following format: `<project-name>/v<version>` Customizable separators and pre/suffixes will be available in the future.

## CLI
1. Run `monotrack init` to create a template configuration (`monotrack.yaml`). The `.monotrack-manifest.yaml` is a work in progress.
2. Edit the config file to match your actual paths and dependencies.
3. Run `monotrack compare <baseSHA> <HEAD>` to list packages that changed

> **_Note:_** Other commands are available but not yet implemented.

## Action
```yaml
- name: Run Monotrack CLI
  id: monotrack
  uses: arnoldvann/monotrack@v0.2.0
  with:
    args: ""                    # Optional
    version: "v0.2.0"           # Optional, defaults to 'latest'
    command: "tag list"         # Optional, defaults to 'compare'
    # Optionally specify a base and head SHA (not used if command != "compare")
    base: ""
    head: ""
    config: "monotrack.yaml"    # Optional, specify config file

- name: Print changed packages
  shell: bash
  run: |
    # Capture the output from Monotrack and display it
    CHANGED_PACKAGES="${{ steps.monotrack.outputs.output }}"
    echo "The following packages have changed:"
    echo "$CHANGED_PACKAGES"
    # Example: run a command for each changed package
    for pkg in $CHANGED_PACKAGES; do
      echo "Processing $pkg..."
      # Replace with a real command, e.g., build or test
      # ./scripts/build.sh $pkg
    done
```

> **_Note:_** The configuration file is required when using the action.

## Configuration example
Given the following `monotrack.yaml`:
```yaml
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

An update to a file in the `packages/another-shared` package will result in the following output:
```bash
$ monotrack compare 8a059ec 0f6a8d1
another-shared
shared-package
backend
```

You can also output bumped versions for specified projects:
```bash
$ git tag --list
frontend/v0.0.1
api/v0.0.2
shared-pkg/v0.0.1
$ monotrack tag bump
frontend/v0.0.2
api/v0.0.3
shared-pkg/v0.0.2
```

Though the default is `patch` and there is currently no way to set version components for specific projects, you can specify a version component:
```bash
$ monotrack tag bump --component minor
frontend/v0.1.0
api/v0.1.0
shared-pkg/v0.1.0
```
The way to work around this is to specify `--projects` and run the `bump` command for each group of projects that can be bumped with the same version component.  

Specify a base commit hash used to diff:
```bash
$ monotrack tag bump bf25f51 -c minor
frontend/v0.1.0
```

# TODO
- [ ] Dynamically generate `monotrack.yaml`  
- [ ] Keep track of versions/tags in the `.monotrack-manifest.yaml`  
- [ ] Implement other helper commands  
- [ ] Support different output formats for the root command (by name, by path, by tag, etc.)  
- [ ] Automatically create git tags if none exist yet? Since tags are required for the VersionBumper to get the commit refs to base the diff on.  
- [ ] Sort outputs alphabetically
- [ ] Implement pre release logic for bump command
