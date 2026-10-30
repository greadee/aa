# Repository media

This folder holds the media referenced by the README. Assets are small,
reviewable, and reproducible. Do not commit large binaries.

| Asset | Purpose | Format | Status |
|---|---|---|---|
| [`terminal-selfcheck.svg`](terminal-selfcheck.svg) | `aa selfcheck` terminal capture | SVG | committed |
| [`console-tui.svg`](console-tui.svg) | Offline `aa console --fake` TUI capture | SVG | committed |
| [`console-web.svg`](console-web.svg) | Read-only web surface content capture | SVG | committed; owner browser screenshot optional |
| [`architecture.svg`](architecture.svg) | Current composition image for the README | SVG | committed |
| [`social-preview.svg`](social-preview.svg) | Source artwork for the GitHub social preview | SVG | source committed; owner upload required |
| `demo-video` | Short product walkthrough | external link | owner action |

## Reproduction

```sh
go build -o /tmp/aa-app ./app
/tmp/aa-app selfcheck
/tmp/aa-app console --fake
```

The terminal captures are deterministic snapshots of those commands. The node id
in `aa selfcheck` changes per run; that is expected and not a stability claim.

The web capture is a content render of the deterministic read-only surface. No
browser is installed in CI, so an owner browser screenshot can replace it without
changing any documentation link.

## Validation

```sh
python tools/docs/check_media.py
```

The check enforces non-empty alt text, existing local targets, a 2 MiB limit, and
well-formed SVG markup. It runs in the `Docs Link Check` workflow.

## Owner actions

1. **Social preview:** export `social-preview.svg` to a 1280×640 PNG/JPG and
   upload it in **Settings → General → Social preview**. GitHub has no public API
   for this setting.
2. **Video:** provide a short walkthrough link, or approve omitting the video
   until Phase 13.

Both actions are recorded on #90 and #86. Documentation links do not depend on
either action.
