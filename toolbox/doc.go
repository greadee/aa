// Package toolbox is aa's extensibility module: the tool, plugin, and MCP
// registry plus the configurable workflow runtime.
//
// It owns tool/plugin/MCP manifests, capability and permission grants,
// sandbox decisions, provider-agnostic invocation, a workflow compiler and
// resumable runtime, and the policy engine. It never grants a capability
// implicitly. See docs/architecture/README.md and the module directive
// aa-toolbox.md.
package toolbox
