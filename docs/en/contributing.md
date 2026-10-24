# Contributing

[한국어](../../CONTRIBUTING.md) | **English** · [English README](../../README.en.md)

Bug reports, feature proposals, documentation and code contributions are welcome in Korean or English. Follow the [code of conduct](../../CODE_OF_CONDUCT.en.md).

## Coordinate a change

Search [GitHub Issues](https://github.com/leechis7/lazyora/issues) first. Discuss substantial changes before implementation, especially Oracle transaction behavior and UI workflows. Maintainers track work in Jira; external contributors do not need a Jira account.

Report security vulnerabilities privately using [Support and security reports](support.md). Never include database passwords, authentication strings, private hostnames or real business data in issues, logs, screenshots or commits.

## Development

Use Go 1.24 or newer and make. The toolchain version in `mise.toml` is optional. The project's default local build targets Linux ARM64; use [the matching build target](build.md) on other systems.

Start from current `dev`. Maintainers use `feature/LORA-<number>-<topic>` after coordinating the Jira number. External contributors can work in a fork and open a PR against `dev`.

```sh
git switch dev
git pull --ff-only origin dev
# Create your topic branch before editing.
go test -race ./...
go vet ./...
make build
```

Format Go changes with `gofmt` and add meaningful checks for behavior changes. Keep Oracle Docker configuration, scripts, sample SQL and real-database tests under `test/oracle/`. UI changes should consider Korean/English, keyboard/mouse, all three themes and 8/16/256/True Color profiles.

## Pull requests

Describe the concrete problem, resulting behavior, related issue, checks performed and remaining limitations. Ordinary changes target **dev**; **main** is for release integration. Korean remains the primary project language, including commit descriptions; conventional prefixes such as `fix:`, `feat:` or `docs:` are accepted. Coordinate wording with the maintainer if needed.

Update applicable documentation, in-app help and the changelog's `Unreleased` section when user behavior changes. Write and update documentation in Korean and English together, with Korean as the default. This includes README files, user/developer guides and release descriptions. Keep corresponding content, versions, commands, shortcuts, verification limits and language links consistent. This does not require a bulk translation of unchanged historical records. CI must pass before merging. Version numbers and release tags are coordinated by the maintainer; agents do not independently decide a major version.

The workflow is dev → feature → dev PR → main release integration → `vX.Y.Z` tag. New user features increase minor; fixes increase patch. Details are in [development and releases (Korean)](../development.md).

Contributions are provided under [AGPL-3.0-only](../../LICENSE). There is currently no separate contributor agreement or signature requirement.
