## Motivation
While there already exist plenty of tools to help with code verisoning, support across languages, specifically for monorepos, may be considered insufficient.  
Monotrack is a generic solution to help ease the pains of CI in a monorepo.  
Monotrack allows you to easily run jobs, create tags, write changelogs and review these changes for projects that have actually changed,  
while taking into consideration local internal dependency trees.

> [!NOTE]
> **Single-project repos** are supported with no extra flags. When `monotrack.yaml` defines exactly one project, monotrack switches behavior automatically: tags are emitted as `vX.Y.Z` (no `<name>/` prefix), and the project's `path` can be `.` or omitted to mean the repo root. The `CHANGELOG.md` is written at the repo root (or at `changelog.path` in the config).

## Release flow

### PR-based, default

`monotrack tag bump` defaults to a **two-phase, release-please-style flow**: it opens a "release PR" containing the generated changelogs, and only creates the actual git tags once that PR is merged. This lets you review changelog wording and version bumps.

#### Phase 1 — propose

When there are new conventional commits since the last release, `tag bump`:

1. Computes per-project version bumps from conventional commits.
2. Writes/updates `CHANGELOG.md` files.
3. Records the proposed tags in `.monotrack-manifest.yaml` under `pending:`.
4. Commits those files to a long-lived **release branch** (default `monotrack/release-<base-branch>`, force-pushed each run).
5. Opens or updates the release PR via the detected forge (GitHub via `gh` CLI; other hosts get a printed compare URL fallback).
6. Prints JSON with `"status": "proposed"` and `"prUrl": …` for each project. **No tags are created in this phase.**

#### Phase 2 — tag

After you merge the release PR, the next `monotrack tag bump` run on the base branch:

1. Reads `.monotrack-manifest.yaml`'s `pending:` list.
2. Pushes the listed tags atomically at HEAD.
3. Prints JSON with `"status": "released"` for each tag.

The manifest file is **not** modified by the tag phase — its `pending:` list stays in place until the next propose phase overwrites it. This keeps history clean (no per-release "clear manifest" commit).

> [!IMPORTANT]
> The manifest file `.monotrack-manifest.yaml` **must be tracked in version control** for the PR flow to work — it is how the propose phase communicates pending tags to the tag phase across CI runs.

#### Editing release-note wording (commit overrides)

The changelog is regenerated on every propose run, so editing `CHANGELOG.md` inside the release PR does not persist. To override the wording for a specific commit, edit the **source commit body** (typically the squash-merge commit message or its PR body) and wrap the override in `BEGIN_COMMIT_OVERRIDE` / `END_COMMIT_OVERRIDE` markers:

```
chore: original boring subject

BEGIN_COMMIT_OVERRIDE
feat(api): user-friendly description of what this actually does

fix(db): another entry attributed to the same commit
END_COMMIT_OVERRIDE
```

On the next propose run, the override is parsed and used in place of the original message. Empty lines inside the block separate entries; each is parsed as its own conventional commit.

> [!NOTE]
> The override works reliably with **squash-merge** (where the PR body becomes the commit body). Plain merges leave the override unattached to a specific commit and are not recommended for repos that rely on this feature.

#### Configuration

```yaml
# monotrack.yaml
release:
  branch: "monotrack/release-{base}"   # default; {base} is replaced with the base branch
  # or a literal like: "release-branch-pr"
```

The `--release-branch <name>` flag overrides this for one invocation.

### Direct-push mode (`--no-pr`)

If you don't want the PR step — e.g. a single-developer repo, RC tags on a staging branch, or a workflow where merges to the target branch are already a review gate — pass `--no-pr` for a direct-commit-and-tag flow:

```bash
monotrack tag bump --no-pr
```

In `--no-pr` mode, `tag bump`:
1. Writes and stages the changelog files.
2. Creates a `chore(release): bump N project(s)` commit on top of `HEAD`.
3. Tags the **new** commit (so the tag's tree contains its own changelog).
4. Atomically pushes the branch and the new tags in one `git push --atomic`.

##### When to pick which mode

| Situation | Mode |
|---|---|
| Single-branch repo, PR is the only review surface for changes | PR mode (default) |
| RC tags on a staging branch driven by every push | `--no-pr` |
| Final release tags created automatically on push to a protected branch you already merge into | `--no-pr` |
| Branch-promotion model (e.g. `main` → `production`) | `--no-pr` on both, review changelog content via the promotion PR diff |

`--dry` never enters the PR or tag-phase routing — it computes proposed bumps and prints them, period. Safe to use in matrix-discovery jobs regardless of which mode the non-dry job uses.

> [!IMPORTANT]
> Monotrack ignores commits whose subject starts with `chore(release)` when deciding which projects need a bump. This prevents the auto-generated release commit from triggering an empty re-release on the next run. Avoid using that prefix for unrelated chore commits if you want them to count toward a bump.

## Changelogs
##### Changelog format

Each bumped project gets a `CHANGELOG.md` written/prepended at its project path. Entries are grouped into **Breaking Changes**, **Features**, **Bug Fixes**, **Performance**, and **Other**. A project that was bumped purely because a dependency changed gets an "Updated internal dependencies" entry.

##### Other changelog flags:

```bash
# write a single combined CHANGELOG.md at the repo root, grouped by project
monotrack tag bump --single-changelog

# write changelog files but DON'T commit them (--no-pr only); tag the original HEAD
monotrack tag bump --no-pr --no-commit-changelog

# skip changelog generation entirely
monotrack tag bump --no-changelog
```

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
    build:
      entrypoint: true
    dependsOn:
      - go-shared
  ui:
    type: node
    path: packages/ui
  go-shared:
    type: go
    path: packages/go-shared
    dependsOn:
      - nested-shared
  nested-shared: # Changes in nested internal deps will be bubbled up to parents
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

### Custom changelog path

When the repo defines a single project, or when `--single-changelog` is passed, the combined `CHANGELOG.md` is written to the repo root by default. Override the location via:

```yaml
changelog:
  path: docs/CHANGELOG.md   # default: CHANGELOG.md
```

In monorepo per-project mode (the default), each changelog still lives at `<project-path>/CHANGELOG.md` and this option is ignored.

### Custom tag scheme

By default tags are formatted as `<name>/v<version>` (monorepo) or `v<version>` (single-project). Both the separator between name and version and the prefix before the version can be customized via the top-level `tags:` block:

```yaml
tags:
  separator: "@"        # default "/"; ignored in single-project mode
  versionPrefix: ""     # default "v"; set to "" explicitly for bare semver tags
```

The example above yields tags like `api@1.2.3`. Tag parsing uses the same scheme, so existing tags from a previous scheme won't be picked up after a change — bump or migrate explicitly.

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

**Optional** CLI version (docker image tag). When unset, defaults to the moving `v<major>` tag derived from the action ref — e.g. `uses: arnoldvann/monotrack@v0.7.3` resolves to image `v0`, which tracks the latest patch of that major. Set explicitly when pinning the action to a branch or SHA; otherwise the action errors out.

### Authentication

The action does not take a token input. Anything that needs to push (`tag bump`) authenticates using the credential `actions/checkout` persisted in the workspace's `.git/config`. To use a token other than the default `GITHUB_TOKEN` (e.g. a GitHub App token that can bypass branch protection), pass it on `actions/checkout`'s `token:` input — see the example below.

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
    uses: arnoldvann/monotrack@v0
    with:
      command: compare
      args: -o json
      config: monotrack.yaml
```

Bump tags for projects that have changed between their latest tag and HEAD.

> [!NOTE]
> The default flow opens a release PR rather than pushing the changelog commit directly, so it works against a protected branch without bypass — the PR goes through normal review/merge. If you pass `--no-pr` to push the changelog commit straight to the branch, the workflow needs an identity that can push to the protected branch: the default `GITHUB_TOKEN` cannot bypass branch protection, so use a GitHub App (or PAT) added to the branch's bypass list and pass its token to `actions/checkout`. Alternatively, pass `--no-pr --no-commit-changelog` so only tags get pushed.

```yaml
    permissions:
      contents: write

    jobs:
      bump:
        runs-on: ubuntu-latest
        outputs:
          projects: ${{ steps.monotrack_json.outputs.projects }}

        steps:
          # Mint a short-lived installation token for the release bot App
          - uses: actions/create-github-app-token@v1
            id: app-token
            with:
              app-id: ${{ vars.RELEASE_BOT_APP_ID }}
              private-key: ${{ secrets.RELEASE_BOT_PRIVATE_KEY }}

          - uses: actions/checkout@v5
            with:
              fetch-depth: 0                                     # Required
              token: ${{ steps.app-token.outputs.token }}        # The push credential

          - name: Configure git user
            run: |
              git config user.name "release-bot[bot]"
              git config user.email "${{ vars.RELEASE_BOT_APP_ID }}+release-bot[bot]@users.noreply.github.com"

          - name: Run Monotrack CLI
            id: monotrack
            uses: arnoldvann/monotrack@v0
            with:
              # version is optional; when unset it tracks the action ref's major (e.g. `v0`)
              command: tag bump               # Optional, defaults to 'tag bump'
              args: -o json --pre-release     # Optional
              config: monotrack.yaml          # Optional

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

For a `--dry` invocation that only computes the project matrix and doesn't push (e.g., gating a build matrix before the real bump), the App token isn't needed — `actions/checkout` with the default `GITHUB_TOKEN` is fine.

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

1. Run `monotrack init` to create a template configuration (`monotrack.yaml`) and an empty `.monotrack-manifest.yaml`. The manifest is read/written by the PR-based release flow; commit it to version control so CI runs can read it across jobs.
2. Edit the config file to match your actual paths and dependencies.
3. Run `monotrack compare --head <HEAD>` to list packages that changed

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

By default, the bump kind for each project is derived from its [Conventional Commit](https://www.conventionalcommits.org/) history since its last tag:

| Commit                               | Bump  |
|--------------------------------------|-------|
| `feat!: …` or `BREAKING CHANGE:` footer | major |
| `feat: …`                            | minor |
| `fix:`, `chore:`, `refactor:`, …     | patch |
| Non-conventional message             | patch (excluded from changelog) |

Commit-to-project mapping is determined by the file-diff logic — the conventional-commit *scope* is informational only and is not used to attribute commits to projects.

To force a single component for every changed project (overriding the derived kind), pass `--component`:
```bash
monotrack tag bump --component minor
frontend/v0.1.0
api/v0.1.0
```

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
- [ ] Sort outputs alphabetically
- [ ] For helm, update versions in each Chart.yaml `version`, and detect umbrella charts in order to update `dependencies[n].version` in the parent
