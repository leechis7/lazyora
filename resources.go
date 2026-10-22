// Package lazyora provides documentation embedded in the standalone executable.
// Keeping the embed declaration here makes the root CHANGELOG the single source
// for both GitHub and the in-app release notes.
package lazyora

import "embed"

// Documents contains the release history and localized, task-oriented guides.
//
//go:embed CHANGELOG.md docs/guide.ko.md docs/guide.en.md
var Documents embed.FS
