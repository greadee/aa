// Package obsv is aa's product-neutral work-observation module: observation
// protocol v1, a bounded failure-open emitter, a journal with cursor
// replay/subscribe/retention, an attach-or-own local transport, deterministic
// work reports, and the Codex `exec` translator.
//
// The kernel hosts it; the visualizer consumes it. It never calls models, owns
// canonical history, or requires a runtime. See docs/architecture/README.md.
package obsv
