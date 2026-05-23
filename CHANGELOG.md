# Changelog

## [0.9.4](https://github.com/ArnoldVanN/monotrack/compare/v0.9.3...v0.9.4) (2026-05-23)


### Bug Fixes

* **changelog:** render accurate fallback line per bump reason ([db2dd10](https://github.com/ArnoldVanN/monotrack/commit/db2dd10b33b8d10a9e8fc75e52bcec0bdb38256b))

## [0.9.3](https://github.com/ArnoldVanN/monotrack/compare/v0.9.2...v0.9.3) (2026-05-23)


### Bug Fixes

* detect history for changelog and bump separately ([f72339c](https://github.com/ArnoldVanN/monotrack/commit/f72339ccb408d0c2cafa58a58581bc99ba657369))

## [0.9.2](https://github.com/ArnoldVanN/monotrack/compare/v0.9.1...v0.9.2) (2026-05-22)


### Bug Fixes

* restore all tags (incl. unreachable) for version selection ([51f90ac](https://github.com/ArnoldVanN/monotrack/commit/51f90acf8ad46dd88b45c107e11497302f00dc21))

## [0.9.1](https://github.com/ArnoldVanN/monotrack/compare/v0.9.0...v0.9.1) (2026-05-22)


### Bug Fixes

* ignore tags not reachable from head ([9aff1af](https://github.com/ArnoldVanN/monotrack/commit/9aff1af055e8d709b50d7e7f9587a4843bddbed6))

## [0.9.0](https://github.com/ArnoldVanN/monotrack/compare/v0.8.3...v0.9.0) (2026-05-21)


### ⚠ BREAKING CHANGES

* promote pre-release tags when -p not specified on tag bump

### Features

* promote pre-release tags when -p not specified on tag bump ([1185355](https://github.com/ArnoldVanN/monotrack/commit/11853555a481b65470aee337705f10b1b1e3dace))

## [0.8.3](https://github.com/ArnoldVanN/monotrack/compare/v0.8.2...v0.8.3) (2026-05-21)


### Bug Fixes

* switch to downloading prebuilt binary in action ([87ffa2a](https://github.com/ArnoldVanN/monotrack/commit/87ffa2abd4282a9c1adc319ca543811ec2adc592))

## [0.8.2](https://github.com/ArnoldVanN/monotrack/compare/v0.8.1...v0.8.2) (2026-05-21)


### Bug Fixes

* action.yaml ACTION_REF possibly undefined ([e221f68](https://github.com/ArnoldVanN/monotrack/commit/e221f68c338c2825fee7d305d7dbaa924e718155))

## [0.8.1](https://github.com/ArnoldVanN/monotrack/compare/v0.8.0...v0.8.1) (2026-05-21)


### Features

* improve PR body ([ee38f6c](https://github.com/ArnoldVanN/monotrack/commit/ee38f6ccbe57ab5b677df772843c6dc0ecb7029e))

## [0.8.0](https://github.com/ArnoldVanN/monotrack/compare/v0.7.1...v0.8.0) (2026-05-21)


### ⚠ BREAKING CHANGES

* add PR-based release flow, added customizability for tags and

### Features

* add PR-based release flow, added customizability for tags and ([72da8e7](https://github.com/ArnoldVanN/monotrack/commit/72da8e74a2ee719621c9544046e2ddddb0409bf4))

## [0.7.1](https://github.com/ArnoldVanN/monotrack/compare/v0.7.0...v0.7.1) (2026-05-09)


### Bug Fixes

* pre-release -&gt; release promotion, rebase on push error when remote changed ([dd63c4e](https://github.com/ArnoldVanN/monotrack/commit/dd63c4e11ab6edf5d13735402407db2880446581))

## [0.7.0](https://github.com/ArnoldVanN/monotrack/compare/v0.6.7...v0.7.0) (2026-05-09)


### ⚠ BREAKING CHANGES

* prevent json output parsing err by outputting debug to stderr,

### Features

* autodetect semver component to bump, write changelogs ([7787397](https://github.com/ArnoldVanN/monotrack/commit/7787397ad4c4a16a19995516b48fcb93f92ea3ab))


### Bug Fixes

* prevent json output parsing err by outputting debug to stderr, ([aa8213c](https://github.com/ArnoldVanN/monotrack/commit/aa8213cefb4df08d7f06bb37a475c483c9ce7497))

## [0.6.7](https://github.com/ArnoldVanN/monotrack/compare/v0.6.6...v0.6.7) (2026-02-02)


### Bug Fixes

* added ProjectTypeHelm to project initialization switch case ([09cd18b](https://github.com/ArnoldVanN/monotrack/commit/09cd18be71e0a8b52f3d3f2add2f0c9dbba6363f))

## [0.6.6](https://github.com/ArnoldVanN/monotrack/compare/v0.6.5...v0.6.6) (2026-02-02)


### Bug Fixes

* remove entrypoint error for helm projects ([0d03888](https://github.com/ArnoldVanN/monotrack/commit/0d03888668a17960074f7cc2cafef9a8f2988600))

## [0.6.5](https://github.com/ArnoldVanN/monotrack/compare/v0.6.4...v0.6.5) (2026-01-23)


### Bug Fixes

* switched from merging diff results to comparing each project ([9997cbe](https://github.com/ArnoldVanN/monotrack/commit/9997cbecebe478bee3369956d9925210fd82c529))

## [0.6.4](https://github.com/ArnoldVanN/monotrack/compare/v0.6.3...v0.6.4) (2026-01-23)


### Bug Fixes

* add id to release-please step ([0cbb654](https://github.com/ArnoldVanN/monotrack/commit/0cbb654005c705a235fe93d477bec8bc5759443a))

## [0.6.3](https://github.com/ArnoldVanN/monotrack/compare/v0.6.2...v0.6.3) (2026-01-23)


### Features

* added project type to json output ([718b44d](https://github.com/ArnoldVanN/monotrack/commit/718b44d6b1bdb7a9bdf30df91cbccbb23c3cc9a8))

## [0.6.2](https://github.com/ArnoldVanN/monotrack/compare/v0.6.1...v0.6.2) (2026-01-23)


### Bug Fixes

* added github token input ([a21e6f8](https://github.com/ArnoldVanN/monotrack/commit/a21e6f8e1c93565165e781e9f080464af4e37723))

## [0.6.1](https://github.com/ArnoldVanN/monotrack/compare/v0.6.0...v0.6.1) (2026-01-23)


### Bug Fixes

* incorrect dependency resolution ([6af321f](https://github.com/ArnoldVanN/monotrack/commit/6af321f34ea771e57fb3405d4f174c8a8faa1e15))
* modified change detection logic ([c960343](https://github.com/ArnoldVanN/monotrack/commit/c960343a5fb5bf2303161947a921032befd0b1c4))

## [0.6.0](https://github.com/ArnoldVanN/monotrack/compare/v0.5.0...v0.6.0) (2026-01-13)


### ⚠ BREAKING CHANGES

* only bump tags for entrypoints

### Features

* only bump tags for entrypoints ([e38417d](https://github.com/ArnoldVanN/monotrack/commit/e38417d0cf960c32fffcc5dfb8a4e7402fb9cd39))


### Bug Fixes

* check error after creating git tag locally ([06b74e0](https://github.com/ArnoldVanN/monotrack/commit/06b74e043bf67fe6a23689b69bd6935db47ae2ca))

## [0.5.0](https://github.com/ArnoldVanN/monotrack/compare/v0.4.24...v0.5.0) (2026-01-13)


### ⚠ BREAKING CHANGES

* create git tags on `tag bump`

### Features

* create git tags on `tag bump` ([83206f1](https://github.com/ArnoldVanN/monotrack/commit/83206f1b60f7291404fd35c760d9be198fa5f3f7))

## [0.4.24](https://github.com/ArnoldVanN/monotrack/compare/v0.4.23...v0.4.24) (2026-01-07)


### Bug Fixes

* incorrect action output ([2f8fecd](https://github.com/ArnoldVanN/monotrack/commit/2f8fecd8b272fdd601d11ccd73be4fca896015d6))

## [0.4.23](https://github.com/ArnoldVanN/monotrack/compare/v0.4.22...v0.4.23) (2026-01-07)


### Bug Fixes

* im blind ([491bbaf](https://github.com/ArnoldVanN/monotrack/commit/491bbaf1f46a86e0f239e12bf59d119397b60745))

## [0.4.22](https://github.com/ArnoldVanN/monotrack/compare/v0.4.21...v0.4.22) (2026-01-06)


### Bug Fixes

* action output ([0eb8724](https://github.com/ArnoldVanN/monotrack/commit/0eb8724e3e03c31e0b68b4436659bfd2384e9274))

## [0.4.21](https://github.com/ArnoldVanN/monotrack/compare/v0.4.20...v0.4.21) (2026-01-06)


### Bug Fixes

* corrected prerelease bump logic ([6ae197c](https://github.com/ArnoldVanN/monotrack/commit/6ae197c75d37e5af1930697871020db353e5f0a0))

## [0.4.20](https://github.com/ArnoldVanN/monotrack/compare/v0.4.19...v0.4.20) (2026-01-06)


### Features

* added logging to action ([854a65a](https://github.com/ArnoldVanN/monotrack/commit/854a65a3dd1aecf92e39f8d58ad34f86a7d33ece))

## [0.4.19](https://github.com/ArnoldVanN/monotrack/compare/v0.4.18...v0.4.19) (2026-01-06)


### Bug Fixes

* formatting of inputs.command and inputs.arg ([c885dcb](https://github.com/ArnoldVanN/monotrack/commit/c885dcbd4d08c00382ff5dd86081b540ee98f7b8))

## [0.4.18](https://github.com/ArnoldVanN/monotrack/compare/v0.4.17...v0.4.18) (2026-01-06)


### Bug Fixes

* action output ([99efafd](https://github.com/ArnoldVanN/monotrack/commit/99efafd9e4e514b06facafb4604755d379a8065e))

## [0.4.17](https://github.com/ArnoldVanN/monotrack/compare/v0.4.16...v0.4.17) (2026-01-06)


### Bug Fixes

* filtering issues with defaulting tags and prereleases ([ad16a80](https://github.com/ArnoldVanN/monotrack/commit/ad16a80576cdf9b3d4ae83f072deedf4de35701c))

## [0.4.16](https://github.com/ArnoldVanN/monotrack/compare/v0.4.15...v0.4.16) (2026-01-06)


### Bug Fixes

* dubious ownership error by adding /repo to safe.directory git conf ([e864f04](https://github.com/ArnoldVanN/monotrack/commit/e864f04a892c574ae834713d2439176705e9e803))

## [0.4.15](https://github.com/ArnoldVanN/monotrack/compare/v0.4.14...v0.4.15) (2026-01-06)


### Bug Fixes

* incorrect error output for GetRepoRoot ([b471b6f](https://github.com/ArnoldVanN/monotrack/commit/b471b6f8f476ef0d93379ef80a8797823acf7f0b))

## [0.4.14](https://github.com/ArnoldVanN/monotrack/compare/v0.4.13...v0.4.14) (2026-01-06)


### Bug Fixes

* log errors in GetRepoRoot ([cacaddd](https://github.com/ArnoldVanN/monotrack/commit/cacaddd1a18cd22f78f6ce1a499ab22e96eb230d))

## [0.4.13](https://github.com/ArnoldVanN/monotrack/compare/v0.4.12...v0.4.13) (2026-01-06)


### Bug Fixes

* added git to docker image ([298da2d](https://github.com/ArnoldVanN/monotrack/commit/298da2d71f188b9eb37e01018b9477a6cc23cca8))

## [0.4.12](https://github.com/ArnoldVanN/monotrack/compare/v0.4.11...v0.4.12) (2026-01-06)


### Features

* added volume mount at GITHUB_WORKSPACE to action docker image ([ed0f614](https://github.com/ArnoldVanN/monotrack/commit/ed0f614657b699f3e73bdde10757c7809c6ef36b))

## [0.4.11](https://github.com/ArnoldVanN/monotrack/compare/v0.4.10...v0.4.11) (2026-01-06)


### Features

* capture non zero exit in action and entrypoint ([1131819](https://github.com/ArnoldVanN/monotrack/commit/113181918f172cd34e925b696351616db6dab159))

## [0.4.10](https://github.com/ArnoldVanN/monotrack/compare/v0.4.9...v0.4.10) (2026-01-06)


### Bug Fixes

* split `COMMAND` input ([7be91cd](https://github.com/ArnoldVanN/monotrack/commit/7be91cd5b06fe08c09af86a5bc12abd4917f3a20))

## [0.4.9](https://github.com/ArnoldVanN/monotrack/compare/v0.4.8...v0.4.9) (2026-01-05)


### Bug Fixes

* unbound vars in action ([5eb7f5d](https://github.com/ArnoldVanN/monotrack/commit/5eb7f5d8e2a1f6e746ab6776016adf09d58f5482))

## [0.4.8](https://github.com/ArnoldVanN/monotrack/compare/v0.4.7...v0.4.8) (2026-01-05)


### Features

* log monotrack command in action ([fc4051d](https://github.com/ArnoldVanN/monotrack/commit/fc4051db5802c1ce736f971356ad3ff11868f5f5))

## [0.4.7](https://github.com/ArnoldVanN/monotrack/compare/v0.4.6...v0.4.7) (2026-01-05)


### Bug Fixes

* move code using github event back to action ([6d30ef2](https://github.com/ArnoldVanN/monotrack/commit/6d30ef2b902df5bab45e3642dd7dbc1b7a07b78d))

## [0.4.6](https://github.com/ArnoldVanN/monotrack/compare/v0.4.5...v0.4.6) (2026-01-05)


### Bug Fixes

* action output ([a5cf472](https://github.com/ArnoldVanN/monotrack/commit/a5cf4729bc901a8e1792f035df2d68c69821f81a))

## [0.4.5](https://github.com/ArnoldVanN/monotrack/compare/v0.4.4...v0.4.5) (2026-01-05)


### Bug Fixes

* changed docker image name ([44444ce](https://github.com/ArnoldVanN/monotrack/commit/44444ce9600a88e3a9cd2bcd86f14f138e8d0a85))

## [0.4.4](https://github.com/ArnoldVanN/monotrack/compare/v0.4.3...v0.4.4) (2026-01-05)


### Features

* add latest tag ([6d309b3](https://github.com/ArnoldVanN/monotrack/commit/6d309b33b478aa564f580d09f376cf0410301c96))

## [0.4.3](https://github.com/ArnoldVanN/monotrack/compare/v0.4.2...v0.4.3) (2026-01-05)


### Bug Fixes

* use composite action instead of docker action for dynamic versions ([c766d89](https://github.com/ArnoldVanN/monotrack/commit/c766d8903e4f6e29ef3a74dc42d362def0e254de))

## [0.4.2](https://github.com/ArnoldVanN/monotrack/compare/v0.4.1...v0.4.2) (2026-01-05)


### Bug Fixes

* typo in dockerfile ([ae95406](https://github.com/ArnoldVanN/monotrack/commit/ae954069f6b7eb1e1feba8f8e929a4297b5a0027))

## [0.4.1](https://github.com/ArnoldVanN/monotrack/compare/v0.4.0...v0.4.1) (2026-01-05)


### Bug Fixes

* added event to docker meta step ([5f5c63d](https://github.com/ArnoldVanN/monotrack/commit/5f5c63d9768ca0c1f4a5173de7f4a7329abeec09))

## [0.4.0](https://github.com/ArnoldVanN/monotrack/compare/v0.3.5...v0.4.0) (2026-01-05)


### ⚠ BREAKING CHANGES

* trigger release

### Miscellaneous Chores

* trigger release ([71f50c1](https://github.com/ArnoldVanN/monotrack/commit/71f50c13d93480513dfebd4cff70bebaa1eeec8c))

## [0.3.5](https://github.com/ArnoldVanN/monotrack/compare/v0.3.4...v0.3.5) (2026-01-05)


### Bug Fixes

* incompatible glibc version in action docker image ([e5f60fb](https://github.com/ArnoldVanN/monotrack/commit/e5f60fb1a55a4790da5f9ac72cbc64145b2e8712))

## [0.3.4](https://github.com/ArnoldVanN/monotrack/compare/v0.3.3...v0.3.4) (2026-01-05)


### Bug Fixes

* typo in golang image tag ([dfb010d](https://github.com/ArnoldVanN/monotrack/commit/dfb010df041c31257474c40795045cf64fa3fbe8))

## [0.3.3](https://github.com/ArnoldVanN/monotrack/compare/v0.3.2...v0.3.3) (2026-01-05)


### Features

* global `head` and `base` flags for simplicity ([5f0a00e](https://github.com/ArnoldVanN/monotrack/commit/5f0a00e329eb0f046bdc69cd55a6116dc3ff4b3d))

## [0.3.2](https://github.com/ArnoldVanN/monotrack/compare/v0.3.1...v0.3.2) (2026-01-02)


### Bug Fixes

* issue with parsing pre-release tags ([9e70efe](https://github.com/ArnoldVanN/monotrack/commit/9e70efe3c9f7263f003b71701f164c744f3f1995))

## [0.3.1](https://github.com/ArnoldVanN/monotrack/compare/v0.3.0...v0.3.1) (2026-01-02)


### Features

* added `entrypoints-only` flag to bump cmd ([254a5bb](https://github.com/ArnoldVanN/monotrack/commit/254a5bb4e3e78cca3173d37faaa241030a8a26bd))
* added json output through flag to bump cmd ([beab19f](https://github.com/ArnoldVanN/monotrack/commit/beab19f0cf687351b486b79acad9f3cb2d785420))


### Bug Fixes

* version cmd doesn't need to be ran inside a git repo ([4d4688c](https://github.com/ArnoldVanN/monotrack/commit/4d4688c2f33a465e8ec734c129d585ea4b9f9903))

## [0.3.0](https://github.com/ArnoldVanN/monotrack/compare/v0.2.1...v0.3.0) (2025-12-30)


### ⚠ BREAKING CHANGES

* change default Action command input to `tag bump`
* output default tag for a project if none exist instead of warning
* return projects paths from compare instead of names

### Features

* change default Action command input to `tag bump` ([6f1eea9](https://github.com/ArnoldVanN/monotrack/commit/6f1eea9ca6710f948f89182ea8517a43da7721f3))


### Bug Fixes

* output default tag for a project if none exist instead of warning ([10792a2](https://github.com/ArnoldVanN/monotrack/commit/10792a2edbec7c38615cc9fab2215cfa1009c359))
* return projects paths from compare instead of names ([0866177](https://github.com/ArnoldVanN/monotrack/commit/08661773c90762aaa2bb6756b8d703453e056a7a))

## [0.2.1](https://github.com/ArnoldVanN/monotrack/compare/v0.2.0...v0.2.1) (2025-12-29)


### Features

* allow specifying a base hash in tag bump command ([9c8284b](https://github.com/ArnoldVanN/monotrack/commit/9c8284b15ed3bded2b352a2ba39a21b6439f68e9))

## [0.2.0](https://github.com/ArnoldVanN/monotrack/compare/v0.1.2...v0.2.0) (2025-12-19)


### ⚠ BREAKING CHANGES

* moved root command to "compare"

### Features

* added tag list command ([0039fff](https://github.com/ArnoldVanN/monotrack/commit/0039fffb58129d18a3330cf66c802b7e88ddaddf))


### Bug Fixes

* projects not being built correctly when manually specified ([0039fff](https://github.com/ArnoldVanN/monotrack/commit/0039fffb58129d18a3330cf66c802b7e88ddaddf))
* Remove branch for push event in publish workflow ([b5ab03f](https://github.com/ArnoldVanN/monotrack/commit/b5ab03f280a794de8377fa566596de227f368d1c))
* reworked action ([b3eb915](https://github.com/ArnoldVanN/monotrack/commit/b3eb915df62c4c516ed60d2b1e6fb0da474d77aa))
* test, added pre-release to to root cmd ([c0d78cd](https://github.com/ArnoldVanN/monotrack/commit/c0d78cd06ea317928052e25187f9e692afb00fb0))
* trigger release ([55b221f](https://github.com/ArnoldVanN/monotrack/commit/55b221fc77a8ca577a9857cc14dbf76f83d52b3e))


### Code Refactoring

* moved root command to "compare" ([0239e4e](https://github.com/ArnoldVanN/monotrack/commit/0239e4e32235f794c8a6a47686c785625d01a8b9))

## [0.1.2](https://github.com/ArnoldVanN/monotrack/compare/v0.1.1...v0.1.2) (2025-12-17)


### Bug Fixes

* trigger release ([b0a6f4c](https://github.com/ArnoldVanN/monotrack/commit/b0a6f4ca5f86be84581faf38a1441eb1176c59ac))

## [0.1.1](https://github.com/ArnoldVanN/monotrack/compare/v0.1.0...v0.1.1) (2025-12-17)


### Bug Fixes

* Remove branch for push event in publish workflow ([b48ef8f](https://github.com/ArnoldVanN/monotrack/commit/b48ef8f57787ebdb4eda292cd77870e811f6c222))

## [0.1.0](https://github.com/ArnoldVanN/monotrack/compare/v0.0.2...v0.1.0) (2025-12-17)


### ⚠ BREAKING CHANGES

* moved root command to "command"

### Features

* added tag list command ([0039fff](https://github.com/ArnoldVanN/monotrack/commit/0039fffb58129d18a3330cf66c802b7e88ddaddf))


### Bug Fixes

* projects not being built correctly when manually specified ([0039fff](https://github.com/ArnoldVanN/monotrack/commit/0039fffb58129d18a3330cf66c802b7e88ddaddf))


### Code Refactoring

* moved root command to "command" ([60d2d67](https://github.com/ArnoldVanN/monotrack/commit/60d2d67dfed5b65edc5200b3846b3b9e8efed82d))

## [0.0.2](https://github.com/ArnoldVanN/monotrack/compare/v0.0.1...v0.0.2) (2025-12-16)


### Bug Fixes

* test, added pre-release to to root cmd ([c0d78cd](https://github.com/ArnoldVanN/monotrack/commit/c0d78cd06ea317928052e25187f9e692afb00fb0))
