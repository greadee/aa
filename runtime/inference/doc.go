// Package inference is the boundary for the model provider/execution service.
//
// The Python `aa-sifter` service is renamed `inference` and reduced to a model
// provider/execution service (rename/rehome only; its routing, budget,
// verification, and context responsibilities are reserved for later issues).
// The Python subproject will live in this directory; the Go runtime invokes it
// over the inter-module RPC boundary, never by import.
//
// This package currently holds only the architectural boundary marker.
package inference
