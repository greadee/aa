# Phase {{ .Number }}: {{ .Objective }}

## Objective

{{ .Objective }}

## Scope

{{ range .Scope }}- {{ . }}
{{ end }}
## Issues

{{ range .Issues }}- [ ] #{{ . }}
{{ end }}
## Dependencies

{{ .Dependencies }}

## Risks / Open Questions

{{ .Risks }}

## Exit Criteria

- [ ] planned functionality implemented
- [ ] relevant tests passing
- [ ] documentation updated
- [ ] no unresolved blocking bugs
- [ ] phase audit completed
- [ ] phase PR reviewed
- [ ] ready to merge to main
