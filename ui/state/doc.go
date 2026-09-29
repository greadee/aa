// Package state holds interface state.
//
// It tracks per-surface navigation, selection, and connection state; it is not
// canonical project state (that is `memory`) and never writes through anything
// but the control-plane API. Scaffolded; delivered in ph10-ui.
package state
