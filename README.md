# Installation

### Build from source
1. Clone the repository.
2. Run:
   ```bash
   go build -o ./monotrack ./main.go
   ```

### Download binary
```bash
curl -LO https://github.com/ArnoldVanN/monotrack/releases/download/v0.1.2/monotrack_Linux_x86_64.tar.gz
tar -xzf monotrack_Linux_x86_64.tar.gz
mv monotrack /usr/local/bin/
```

# Usage

## CLI
1. Run `monotrack init` to create a template configuration (`monotrack.yaml`). The `.monotrack-manifest.yaml` is a work in progress.
2. Edit the config file to match your actual paths and dependencies.
3. Run `monotrack compare <baseSHA> <HEAD>` to list packages that changed

> **_Note:_** Other commands are available but not yet implemented.

## Action
```yaml
- name: Run Monotrack CLI
  id: monotrack
  uses: arnoldvann/monotrack@v0.1.2
  with:
    args: ""                    # Optional
    version: "v0.1.2"           # Optional, defaults to 'latest'
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
$ monotrack compare c4688b6a4aa2d3a50a0e1ec59c69d0eeacee36b6 428cb452e22252ebee05e6ee8209175f330b16aa
another-shared
shared-package
backend
```

# TODO
- [ ] Dynamically generate `monotrack.yaml`
- [ ] Keep track of versions/tags in the `.monotrack-manifest.yaml`
- [ ] Implement other helper commands
- [ ] Support different output formats for the root command (by name, by path, by tag, etc.)
