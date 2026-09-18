// Package visualizer is aa's time-travel work history and 3D graph module.
//
// It projects the shared event taxonomy into a graph, lays that graph out
// deterministically, and replays it. Live and replay share one projection and
// one render path. It never defines its own observation protocol, never stores
// canonical history, and never imports kernel. See docs/architecture/README.md
// and the module directive aa-visualizer.md.
package visualizer
