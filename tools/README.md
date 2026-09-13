# aa tools

Development and CI tooling for the aa monorepo. Not a product module; it is excluded from the product layering rules.

## archtest

Enforces the module layering from [architecture decision D5](../docs/architecture/README.md#42-layering-and-dependency-rules).

```sh
cd tools
go run ./cmd/archtest -root ..
go test ./...
```

`go test ./...` includes `TestWorkspaceBoundaries`, which scans the real workspace, plus unit tests that prove a violation is detected.
