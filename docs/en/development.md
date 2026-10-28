# Development and publishing

[한국어](../development.md) | **English** · [Contributing](contributing.md)

`dev` integrates development; `main` is the release/public documentation branch. Start feature, fix and documentation work from current `dev`. Maintainers check the related Jira issue before work, update it when work starts and record implementation/verification evidence. External contributors do not need Jira accounts.

## Jira Kanban workflow

Since 2026-10-27, LazyOra uses Kanban without sprints. Prioritize the backlog and use **To Do → In Progress → Review → Done** instead of inventing deadlines or creating new sprints. Preserve the historical records of sprints 1 and 2; sprint 2 is closed.

Work on one issue at a time. Move it to In Progress when work actually starts, then to Review after implementation and required verification, recording evidence and any pending CI/publication in a comment. Move to Done only after explicit user review approval. Return requested rework to In Progress and resubmit it for Review after verification. If Jira has no Review status yet, keep In Progress with a review-pending comment rather than using Done. Keep epics open while required child work remains. Leave work awaiting a dependency, such as a published Tuivory package, in To Do with a recorded reason. Existing commit/publishing approval and sequential-branch rules still apply.

For the team-managed Jira board, disable Sprints under project/space settings → Features. The connected tools do not expose this toggle, so UI configuration still needs confirmation; adopting the workflow and completing the Jira UI setting are separate steps. See [Atlassian’s instructions](https://support.atlassian.com/jira-software-cloud/docs/enable-sprints/).

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

The batch originally allowed Done after implementation and required local checks with evidence. Since 2026-10-27, the Kanban Review step above supersedes that rule: submit verified work for Review and mark Done only after explicit user review approval, distinguishing pending remote CI/publication. Do not close an epic while required child work remains.

## Releases

New user features increase minor and reset patch; fixes increase patch, including under 0.x. Only the user decides a major/product-version change. The maintainer coordinates version/tag decisions.

1. Prepare a release branch from dev. Update `internal/appinfo/info.go`, **both** README versions and changelog compare links. Move `Unreleased` entries to the dated version and prepare Korean/English release descriptions.
2. Integrate through dev and create the dev → main PR.
3. After CI and required verification, merge using a **merge commit**, preserving ancestry. Do not squash/rebase normal dev → main release integration.
4. At the user's release request, create a new annotated `vX.Y.Z` tag on a commit included in main and push the tag. Never overwrite existing tags/releases.

Release Actions validate tag syntax, app-version equality and main ancestry; run race/vet and six-target builds; then package executables, LICENSE, README, tview license and SHA256SUMS. The release notes file `docs/releases/vX.Y.Z.md` supplies the GitHub release body when present. Include an English section or link to its paired English description. Main merge alone does not publish a release.

After publication, verify assets/checksums and the actual executable version, synchronize main's merge/history back into dev using a merge commit, and clean branches. Historical releases remain dated evidence.

## Dependencies and UI

Published LazyOra v0.11.1 pins **Tuivory v0.1.0**, `github.com/leechis7/tuivory/go`, including its public tree component. LazyOra v0.11.0 includes the new tree. Develop Tuivory itself separately. Review public versions/commits and import the module; do not copy it or replace it with a local checkout. Record updates in [Tuivory integration](tuivory-sync.md) and use [UI adapters](ui-style.md).

The tview v0.42.0 local fork adds SQL TextArea highlighting. Its source difference is confined to `textarea.go`; retain the upstream MIT notice and rendering checks. Workflow Actions are pinned to commit SHAs; Go toolchain versions are coordinated with `mise.toml`. Module downloads retry transient failures, but actual check failures remain failures.

Branch rules can require the `Test, vet and build` check; workflow files alone do not enforce merge protection.

## Korean and English together

Korean is primary. New/changed README, user/developer guidance and release descriptions are updated in both languages. Keep versions, commands, shortcuts, verification scope and language links aligned. English user docs live under `docs/en/`; the offline user guide is `docs/guide.en.md`. Unchanged historical evidence does not require bulk translation. Use the [document index](index.md) to distinguish live guidance from archives and proposals.

## Prefer Tuivory for shared UI

Check and use public Tuivory APIs first for shared drawing, borders/overlap repair, controls and styles. Do not maintain independent fixes for the same defect in both projects. Define missing shared capabilities in the separate Tuivory project, then integrate published APIs through LazyOra adapters. Oracle execution/transactions, tab/workspace data ownership, commands and language/ASCII preference policy remain product responsibilities. Style tview widgets with Tuivory Styler; preserve input, focus, modal isolation, copying and data safety when adopting new models/views.

The development branch pins public `github.com/leechis7/tuivory/go v0.2.0`, tag `go/v0.2.0`, source `987d1a6786033a6bf063235b5e7fea7b565f0af0`. Delegate the whole popup draw to `WideSafeScreen` and clipped form/button labels to `CellText`; remove local `popupFill`. Published v0.11.1 still contains v0.1.0; a development dependency update does not change released binaries. See [adoption/defer review](../audits/lora-76-tuivory.md).
