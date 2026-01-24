## Purpose
While there already exist plenty of tools to help with code verisoning, support across languages, specifically for monorepos, may be considered insufficient.  
Monotrack is a generic and simple solution to help ease the pains of CI in a monorepo.  
This project is currently in it's testing phase and has some limitations. For example having to manually specify a local/internal dependency tree.  
Though this is a limitation other CI tools also observe.  
For now, the main use cases are easily running jobs and creating tags for projects that have actually changed,  
while taking into consideration local internal dependency trees.

## Configuration example
Given the following `monotrack.yaml`:
```yaml
projects:
  web:
    type: node
    path: apps/web
    build:
      entrypoint: true # Only entrypoints will be included in tag bumps
    dependsOn:
      - ui
  docs:
    type: node
    path: apps/docs
    build:
      entrypoint: true
    dependsOn:
      - ui
  api:
    type: go
    path: apps/api
    dependsOn:
      - go-shared
    build:
      entrypoint: true
  go-shared:
    type: go
    path: packages/go-shared
    dependsOn:
      - nested-shared
  ui:
    type: node
    path: packages/ui
  nested-shared:
    type: go
    path: packages/nested-shared
```

An update to a file in the `packages/nested-shared` package will result in the following output:
```bash
monotrack compare --head 34c2818
api
nested-shared
go-shared
```

Or, when bumping, since only entrypoints are bumped:
```bash
monotrack tag bump --dry
api/v0.0.2
```

## Action

## Inputs

### `command`

**Optional** The command to run. Defaults to `tag bump`

### `base`

**Optional** The base SHA used to compare. Will use the commit referenced in the latest tag if unspecified.

### `head`

**Optional** The HEAD SHA used to compare, might be required in special cases, like when triggering a workflow manually

### `args`

**Optional** Arguments to pass to the command

### `config`

**Optional** Name of the configuration file

### `version`

**Optional** CLI version. Defaults to latest

### `token`

**Optional** GitHub token used to push tags. Required when using `tag bump`

## Outputs

### `output`

The output of the CLI

## Example usage
Perform operations on all projects including internal dependencies:
```yaml
  - uses: actions/checkout@v5
    with:
      fetch-depth: 0 # Required

  - name: Run Monotrack CLI
    id: monotrack
    uses: arnoldvann/monotrack@v0.6
    with:
      version: v0.6.5
      command: compare
      args: -o json
      config: monotrack.yaml
```

Bump tags for projects that have changed between their latest tag and HEAD.
```yaml
    # Required for tag creation
    permissions:
      contents: write

    jobs:
      bump:
        runs-on: ubuntu-latest
        outputs:
          projects: ${{ steps.monotrack_json.outputs.projects }}

        steps:
          - uses: actions/checkout@v5
            with:
              fetch-depth: 0 # Required

          - name: Run Monotrack CLI
            id: monotrack
            uses: arnoldvann/monotrack@v0.6
            with:
              version: v0.6.5                 # Optional, defaults to 'latest' (recommended)
              command: tag bump               # Optional, defaults to 'tag bump'
              args: -o json --pre-release     # Optional
              config: monotrack.yaml          # Optional
              token: ${{ github.token }}

          - name: Output monotrack result
            id: monotrack_json
            shell: bash
            run: |
              OUTPUT='${{ steps.monotrack.outputs.output }}'
              echo "projects<<EOF" >> "$GITHUB_OUTPUT"
              echo "$OUTPUT" >> "$GITHUB_OUTPUT"
              echo "EOF" >> "$GITHUB_OUTPUT"

      # Do something with the output like build, test, release, etc
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
          type: ${{ matrix.type }}
```

> [!TIP]  
> For a full working example, see the [testing repo](https://github.com/ArnoldVanN/monotrack-testing)

## CLI

## Installation

### Build from source
1. Clone the repository.
2. Run:
```bash
go build -o ./monotrack ./main.go
```

### Download binary
```bash
curl -LO https://github.com/ArnoldVanN/monotrack/releases/download/v0.6.2/monotrack_Linux_x86_64.tar.gz
tar -xzf monotrack_Linux_x86_64.tar.gz
mv monotrack /usr/local/bin/
```

1. Run `monotrack init` to create a template configuration (`monotrack.yaml`). The `.monotrack-manifest.yaml` is a work in progress.
2. Edit the config file to match your actual paths and dependencies.
3. Run `monotrack compare --head <HEAD>` to list packages that changed

> [!NOTE]  
> Other commands might be available in the CLI but are not yet implemented.  

### Examples

#### Bump git tags for specified projects
```bash
git tag --list
frontend/v0.0.1
api/v0.0.2

monotrack tag bump
frontend/v0.0.2
api/v0.0.3
```

> [!IMPORTANT]  
> If no git tags matching a project specified in the config exist, `tag` commands will default to `<project>/v0.0.0`  

Though the default is `patch` and there is **currently** no way to set version components for specific projects, you can specify a version component:
```bash
monotrack tag bump --component minor
frontend/v0.1.0
api/v0.1.0
```
The way to work around this is to specify `--projects` and run the `bump` command for each group of projects that can be bumped with the same version component.  
PR's to base version component on git commit messages are welcome...

#### Output as json
```bash
monotrack tag bump --head 8a059ec --projects api -o json
[{"name":"api","path":"apps/api","version":"v0.0.3","type":"go"}]
```

#### Use prereleases
```bash
monotrack tag bump --head 8a059ec --projects api -o json -p
[{"name":"api","path":"apps/api","version":"v0.0.4-rc.1","type":"go"}}]
```

#### Run the bump command without making any changes (dry run)
```bash
monotrack tag bump --dry
frontend/v0.0.1
api/v0.0.1
shared-pkg/v0.0.1
```

# TODO
- [ ] Dynamically generate `monotrack.yaml`  
- [ ] Keep track of versions/tags in the `.monotrack-manifest.yaml`  
- [ ] Sort outputs alphabetically
- [ ] Base version bump component on git commit message history? (aka make a release-please clone for monorepos)
- [ ] Add tests
- [ ] Default action inputs.version to latest patch of current action major
