# {{ .Title }}

## Goal

{{ .Goal }}

## Requirements

{{ range .Requirements }}- {{ . }}
{{ end }}
## Acceptance Criteria

{{ range .Acceptance }}- [ ] {{ . }}
{{ end }}
- [ ] observable behavior
- [ ] tests added or updated
- [ ] error cases handled
- [ ] documentation updated where required

## Notes

{{ .Notes }}
