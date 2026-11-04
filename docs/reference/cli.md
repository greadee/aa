# CLI reference

The `aa` binary is the composition root and the console entry point. The source
is [`app/main.go`](../../app/main.go).

## Commands

| Command | Purpose |
|---|---|
| `aa` | Run `selfcheck` |
| `aa selfcheck` / `--selfcheck` / `-c` | Build the graph and exercise each seam deterministically |
| `aa serve` | Run the loopback HTTP/JSON control plane |
| `aa console` | Run the text console against a control plane |
| `aa version` / `--version` / `-v` | Print the version |
| `aa help` / `--help` / `-h` | Print usage |

## `aa serve`

| Flag | Default | Meaning |
|---|---|---|
| `--addr` | `127.0.0.1:7100` | Loopback listen address |
| `--token` | `$AA_CONTROL_TOKEN` | Bearer token required on every path except `/api/v1/health` when set; empty disables the check |
| `--root` | `$AA_ROOT` or `.aa` | Durable state root |

## `aa console`

| Flag | Default | Meaning |
|---|---|---|
| `--control` | `http://127.0.0.1:7100/api/v1` | Control-plane base URL |
| `--token` | `$AA_CONTROL_TOKEN` | Bearer token for the control plane |
| `--project` | `default` | Project id |
| `--web` | off | Serve the read-only web surface instead of the text surface |
| `--fake` | off | Use the deterministic fake control plane; no server or network |
| `--addr` | `127.0.0.1:7101` | Web listen address when `--web` is set |

### Text console keys

| Input | Meaning |
|---|---|
| `j`, `]`, Down, Right, Tab | Next view |
| `k`, `[`, Up, Left | Previous view |
| `:open <view>` | Jump to `projects`, `graph`, `memory`, `nodes`, `tools`, `reports`, or `approvals` |
| `:project <id>` | Switch project and refresh the read models |
| `?` | Show help |
| `q`, `exit`, `:quit`, `:exit` | Quit |

The text surface is line-oriented: arrow keys/Tab and commands are submitted
with Enter.

## Environment

| Variable | Meaning |
|---|---|
| `AA_CONTROL_TOKEN` | Default bearer token for `serve` and `console` |
| `AA_ROOT` | Default state root for `serve` |

## Examples

```sh
aa selfcheck
AA_CONTROL_TOKEN=dev-token aa serve
aa console --token dev-token
aa console --fake
aa console --web --token dev-token
aa console --web --fake
curl http://127.0.0.1:7100/api/v1/health
```

## Related

- [Repository README](../../README.md)
- [UI module](../modules/ui/README.md)
- [Control-plane OpenAPI](../../contracts/openapi/control-plane-v1.yaml)
