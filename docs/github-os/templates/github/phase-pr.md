## Phase Objective

{{ .Objective }}

## Tracking

Phase issue: #{{ .TrackingIssue }}
Milestone: {{ .Milestone }}

## Progress

### Features

{{ range .Features }}- [ ] {{ . }}
{{ end }}
### Infrastructure

{{ range .Infrastructure }}- [ ] {{ . }}
{{ end }}
### Testing

{{ range .Testing }}- [ ] {{ . }}
{{ end }}
### Documentation

{{ range .Documentation }}- [ ] {{ . }}
{{ end }}
## Child Pull Requests

{{ range .ChildPRs }}- [ ] #{{ . }}
{{ end }}
## Current Status

{{ .Status }}

## Known Issues / Risks

{{ .Risks }}

## Validation

- [ ] full test suite passes
- [ ] lint/type checks pass
- [ ] architecture documentation updated
- [ ] audit complete
- [ ] blocking issues resolved

## Exit Criteria

- [ ] all required phase issues complete
- [ ] no open P0/P1 bugs
- [ ] phase code review complete
- [ ] repository audit complete
- [ ] ready for main
