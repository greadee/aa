// Package models is the durable model-definition store.
//
// A model is an inference configuration independent of roles: provider/runtime,
// identifier/version, capabilities, context limits, tool support, locality,
// availability, cost, and latency. The registry stores and serves contracts v2
// ModelSpecs; it performs no allocation or execution. DefaultCatalog mirrors the
// inference service's advisory catalog so the model allocator has a populated
// registry to select from.
package models
