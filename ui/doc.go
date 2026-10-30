// Package ui is aa's unified interface: the CLI and TUI surfaces over the
// control plane, built on pluggable surfaces so a web or desktop surface can be
// added later.
//
// The CLI, TUI, and read-only web surfaces are delivered over the
// control-plane client, including the offline fake client. The desktop surface
// is deferred to v2. See docs/current-state.md and docs/architecture/README.md.
package ui
