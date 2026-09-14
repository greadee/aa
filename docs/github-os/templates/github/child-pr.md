## Summary

{{ .Summary }}

## Related Issue

Closes #{{ .Issue }}

## Changes

{{ range .Changes }}- {{ . }}
{{ end }}
## Validation

- [ ] unit tests
- [ ] integration tests
- [ ] lint/type checks
- [ ] manual verification if relevant

## Notes

{{ .Notes }}

## Phase

Part of #{{ .PhaseIssue }}

Milestone: {{ .Milestone }}
