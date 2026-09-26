// Package runtime owns execution mechanics for allocated work.
//
// It instantiates and runs workers (Role × Model + bindings), exposes the
// provider-neutral worker-execution adapter, and is the integration point for
// sandboxing and the inference service. It does not decide allocation or
// scheduling: the kernel allocator produces an execution plan and the kernel
// scheduler dispatches it here.
//
// Subpackages:
//
//   - `worker`   — the provider-neutral execution adapter and deterministic fake;
//   - `lifecycle` — worker lifecycle (start, stop, termination, events), reserved;
//   - `sandbox`  — process/workload isolation, reserved and not implemented;
//   - `inference` — the Python model provider/execution service (renamed from
//     sifter), a nested subproject.
package runtime
