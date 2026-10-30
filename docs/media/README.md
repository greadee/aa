# Repository media

This folder holds the media referenced by the README. Assets are small,
reviewable, and reproducible. Do not commit large binaries.

| Asset | Purpose | Format | Status |
|---|---|---|---|
| `social-preview.svg` | Source artwork for the GitHub social preview | SVG | source committed |
| `terminal-selfcheck.svg` | Deterministic `aa selfcheck` terminal capture | SVG or PNG | generated in Sprint 3 |
| `console-web.png` | Read-only web console over the fake control plane | PNG | generated in Sprint 3 |
| `architecture.svg` | Current composition diagram for the README | SVG | generated in Sprint 3 |
| `demo-video` | Short product walkthrough | external link | owner action |

## Social preview owner action

GitHub's social preview upload has no public API. The owner must upload a
1280×640 PNG/JPG export of `social-preview.svg` in repository settings:

1. Open **Settings → General → Social preview**.
2. Upload the exported image.
3. Record the upload date on the sprint issue.

This is the only media step that cannot be completed from CI.
