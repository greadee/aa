// Package sync is aa's transport module: remote sync, offline file sharing,
// and parallelization across machines.
//
// It provides peer identity and pairing, a transport seam with a deterministic
// in-memory fake, resumable hash-verified chunked transfer, one-way revision
// sync with tombstones, an offline change queue, work-package and artifact
// distribution, parallel coordination, and the sync.* RPC adapter. It never
// gains or grants execution authority; it moves work only.
package sync
