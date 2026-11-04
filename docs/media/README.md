# Repository media needed

Product screenshots, GIFs, and demo video are **not committed yet**. The owner
will provide the final assets.

## Requested assets

| Asset | Purpose | Suggested format | Alt text |
|---|---|---|---|
| `terminal-selfcheck.png` | `aa selfcheck` output | PNG or GIF | Terminal capture of the deterministic aa selfcheck composition output |
| `console-tui.png` | Offline TUI | PNG or GIF | Terminal capture of the offline aa console text surface |
| `console-web.png` | Read-only web console | PNG | Read-only aa web console showing projects, work, nodes, and tools |
| `demo.gif` or video | Short walkthrough | GIF or external link | Short aa product walkthrough |
| `social-preview.png` | GitHub social preview | 1280×640 PNG/JPG | aa wordmark and tagline on a dark background |

## Capture commands

```sh
go build -o /tmp/aa-app ./app
/tmp/aa-app selfcheck
/tmp/aa-app console --fake
/tmp/aa-app console --web
```

## Owner actions

1. Provide the terminal, TUI, and web captures.
2. Provide a short walkthrough link or GIF, or approve omitting it.
3. Upload a 1280×640 social preview in **Settings → General → Social preview**.
4. Commit the assets under `docs/media/` and update the README media section.
