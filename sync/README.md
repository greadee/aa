# aa-sync

Remote sync, offline file sharing, and parallelization across machines.

- **Owns:** paired mutual-TLS transport; resumable verified chunked transfer; one-way revision sync; offline change queue and reconciliation; work-package and artifact distribution; parallelization transport.
- **Must not:** become a remote shell or gain execution authority.
- **Status:** scaffolded. The working sync MVP migrates from the existing `syncgate` source in `ph6-sync`.
- **Directive:** [aa-sync.md](aa-sync.md)
- **Architecture:** [../docs/architecture/README.md](../docs/architecture/README.md)
