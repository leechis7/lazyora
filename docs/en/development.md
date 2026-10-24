# Development and publishing

[한국어](../development.md) | **English** · [Contributing](contributing.md)

`dev` integrates development; `main` is the release/public documentation branch. Start feature, fix and documentation work from current `dev`. Maintainers check the related Jira issue before work, update it when work starts and record implementation/verification evidence. External contributors do not need Jira accounts.

## Branches and checks

```sh
git switch dev
git pull --ff-only origin dev
git switch -c feature/LORA-123-topic
```

The number is illustrative; use the actual issue agreed with the maintainer. Work through one branch at a time, including review, authorized commits/merges and branch cleanup. Do not switch to unrelated work unless instructed. Commits and GitHub publishing require explicit user approval; commit descriptions use Korean.

The local default `make build` produces Linux ARM64 only. Build other local targets only when requested. CI/Release use `make build-all` for Linux, Windows and macOS AMD64/ARM64. See [build instructions](build.md).

After checks, publish the branch and open a **dev-targeted PR**. CI runs full race/vet and six-target builds, Windows CLI, and Windows/macOS simulated UI/settings/export checks. macOS settings checks use `macos-15-intel`; both macOS architectures are still cross-built. CI artifacts are retained for seven days and may need executable permissions restored after download.

Merge after CI succeeds, then remove completed feature branches. Publishing documentation to the repository's default view also requires main integration; changing only dev does not update the default README. Documentation-only publishing need not create a version tag or rebuild a public release.

## Approved batch exception

The 2026-10-21 approved backlog batch used sequential local implementation/checks/commits/dev merges and branch cleanup, followed by one integration publication/CI phase. This is an explicit exception for that batch, not permission to skip normal CI for all future work. Preserve already-running CI and local dev commits ahead of origin; do not overwrite them with remote state.

Under the updated completion rule, completed implementation and required local checks may be marked Done with evidence, distinguishing pending remote CI/publication. Do not close an epic while required child work remains.

## Releases

New user features increase minor and reset patch; fixes increase patch, including under 0.x. Only the user decides a major/product-version change. The maintainer coordinates version/tag decisions.

1. Prepare a release branch from dev. Update `internal/appinfo/info.go`, **both** README versions and changelog compare links. Move `Unreleased` entries to the dated version and prepare Korean/English release descriptions.
2. Integrate through dev and create the dev → main PR.
3. After CI and required verification, merge using a **merge commit**, preserving ancestry. Do not squash/rebase normal dev → main release integration.
4. At the user's release request, create a new annotated `vX.Y.Z` tag on a commit included in main and push the tag. Never overwrite existing tags/releases.

Release Actions validate tag syntax, app-version equality and main ancestry; run race/vet and six-target builds; then package executables, LICENSE, README, tview license and SHA256SUMS. The release notes file `docs/releases/vX.Y.Z.md` supplies the GitHub release body when present. Include an English section or link to its paired English description. Main merge alone does not publish a release.

After publication, verify assets/checksums and the actual executable version, synchronize main's merge/history back into dev using a merge commit, and clean branches. Historical releases remain dated evidence.

## Dependencies and UI

Common styling uses **Tuivory v0.0.1**, `github.com/leechis7/tuivory/go`, first shipped in v0.10.2. Develop Tuivory itself separately. Review public versions/commits and import the module; do not copy it or replace it with a local checkout. Record updates in [Tuivory integration](tuivory-sync.md) and use [UI adapters](ui-style.md).

The tview v0.42.0 local fork adds SQL TextArea highlighting. Its source difference is confined to `textarea.go`; retain the upstream MIT notice and rendering checks. Workflow Actions are pinned to commit SHAs; Go toolchain versions are coordinated with `mise.toml`. Module downloads retry transient failures, but actual check failures remain failures.

Branch rules can require the `Test, vet and build` check; workflow files alone do not enforce merge protection.

## Korean and English together

Korean is primary. New/changed README, user/developer guidance and release descriptions are updated in both languages. Keep versions, commands, shortcuts, verification scope and language links aligned. English user docs live under `docs/en/`; the offline user guide is `docs/guide.en.md`. Unchanged historical evidence does not require bulk translation. Use the [document index](index.md) to distinguish live guidance from archives and proposals.
