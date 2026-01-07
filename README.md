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
curl -LO https://github.com/ArnoldVanN/monotrack/releases/download/v0.4.20/monotrack_Linux_x86_64.tar.gz
tar -xzf monotrack_Linux_x86_64.tar.gz
mv monotrack /usr/local/bin/
```

# Usage

## Configuration example
Given the following `monotrack.yaml`:
```yaml
projects:
  frontend:
    type: node
    path: apps/frontend
    build:
      entrypoint: true # Optionally specify entrypoints in order to use `-e`. Useful in CI when only needing to build entrypoint projects while testing all of them
  backend:
    type: go
    path: apps/backend
    dependsOn:
      - shared-package
    build:
      entrypoint: true
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
monotrack compare --base 8a059ec --head 0f6a8d1
packages/shared-pkg
apps/backend
apps/frontend
```

## Action

## Inputs

### `command`

**Optional** The command to run. Defaults to `tag bump`

### `base`

**Optional** The base SHA used to compare. Will use the commit referenced in the latest tag if unspecified.

### `head`

**Optional** The HEAD SHA used to compare

### `args`

**Optional** Arguments to pass to the command

### `config`

**Optional** Name of the configuration file

### `verison`

**Optional** CLI version. Defaults to latest

## Outputs

### `output`

The output of the CLI

## Example usage
```yaml
    jobs:
      bump:
        runs-on: ubuntu-latest
        outputs:
          projects: ${{ steps.monotrack_json.outputs.projects }}

        steps:
          - uses: actions/checkout@v5
            with:
              fetch-depth: 0

          - name: Run Monotrack CLI
            id: monotrack
            uses: arnoldvann/monotrack@v0.4.24
            with:
              version: v0.4.24                # Optional, defaults to 'latest'
              command: tag bump               # Optional, defaults to 'tag bump'
              args: -o json --pre-release     # Optional
              # Optionally specify a base and head SHA
              base: ""
              head: ""
              config: monotrack.yaml          # Optional

          - name: Output monotrack result
            id: monotrack_json
            shell: bash
            run: |
              OUTPUT='${{ steps.monotrack.outputs.output }}'
              echo "projects<<EOF" >> "$GITHUB_OUTPUT"
              echo "$OUTPUT" >> "$GITHUB_OUTPUT"
              echo "EOF" >> "$GITHUB_OUTPUT"

      # Do something with the output like build, test, etc
      build:
        needs:
          - bump
        strategy:
          matrix: # Since we set --output to json, we can create a matrix based on that here
            include: ${{ fromJson(needs.bump.outputs.projects) }}
        uses: ./.github/workflows/build.yaml
        with:
          app: ${{ matrix.name }}
          path: ${{ matrix.path }}
          version: ${{ matrix.version }}
        secrets: inherit
```

> [!IMPORTANT]  
> The monotrack configuration file should already exist when using the action.

## CLI
1. Run `monotrack init` to create a template configuration (`monotrack.yaml`). The `.monotrack-manifest.yaml` is a work in progress.
2. Edit the config file to match your actual paths and dependencies.
3. Run `monotrack compare <baseSHA> <HEAD>` to list packages that changed

> [!NOTE]  
> Other commands might be available in the CLI but are not yet implemented.  

### Examples

#### Output bumped versions for specified projects:
```bash
git tag --list
frontend/v0.0.1
api/v0.0.2
shared-pkg/v0.0.1
monotrack tag bump
frontend/v0.0.2
api/v0.0.3
shared-pkg/v0.0.2
```

> [!IMPORTANT]  
> If no git tags matching a project specified in the config exist, `tag` commands will default to `<project>/v0.0.0`  

Though the default is `patch` and there is currently no way to set version components for specific projects, you can specify a version component:
```bash
monotrack tag bump --component minor
frontend/v0.1.0
api/v0.1.0
shared-pkg/v0.1.0
```
The way to work around this is to specify `--projects` and run the `bump` command for each group of projects that can be bumped with the same version component.  

#### Specify a base commit hash used to diff:
```bash
monotrack tag bump --base bf25f51 -c minor
frontend/v0.1.0
```

> [!IMPORTANT]  
> If no commit hashes are specified, they will be derived from the context  

#### Output as json (Only on `bump` currently)
```bash
monotrack tag bump --base 8a059ec --projects api -o json
[{"name":"api","path":"apps/api","version":"v0.0.3"},{"name":"go-shared","path":"packages/go-shared","version":"v0.0.2"}]
```

#### Only list entrypoints (Only on `bump` currently)
```bash
monotrack tag bump --base 8a059ec --projects api -o json -e
[{"name":"api","path":"apps/api","version":"v0.0.3"}]
```

#### Use prereleases
```bash
monotrack tag bump --base 8a059ec --projects api -pe -o json -c minor
[{"name":"api","path":"apps/api","version":"v0.1.0-rc"}]
```

# TODO
- [ ] Dynamically generate `monotrack.yaml`  
- [ ] Keep track of versions/tags in the `.monotrack-manifest.yaml`  
- [ ] Sort outputs alphabetically
- [ ] Create git tags on `tag bump`
