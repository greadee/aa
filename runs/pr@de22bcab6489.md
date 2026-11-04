## Review Summary

_Profile: `pr` · 10 finding(s) · generated 2026-11-04T19:20:00Z_

### Blocking

- None

### Non-Blocking

- **[P2] isContractObject accepts malformed or unsupported contract objects** — `contracts/typescript/v2/index.ts:507`
  - Impact: This type guard only checks for the presence of 'kind' and 'id'. It returns true for objects such as '{ kind: 123, id: null }', objects with an unknown 'kind', objects missing the required 'contractVersion', and objects carrying a different contract version. Callers that narrow untrusted JSON to 'ContractObject' can then access required fields like 'sequence', 'status', or 'steps' that do not exist, causing runtime failures or bypassing version checks.
  - Recommendation: Validate required fields at runtime: at minimum check 'typeof kind === "string"', 'typeof id === "string"', 'contractVersion === CONTRACT_VERSION', and that 'kind' belongs to the supported set. For untrusted input, use a schema validator; otherwise rename the helper to something weaker such as 'isObjectWithKindAndId'.
- **[P2] Template render errors are reported after the response body has already been streamed** — `ui/surfaces/web/web.go:47`
  - Impact: Execute writes directly to the ResponseWriter, so the 200 status and partial HTML are already flushed when an error occurs. The subsequent http.Error cannot change the status code and appends an error string to the truncated HTML, producing a 200 response with malformed markup that clients and tests will treat as success.
  - Recommendation: Render into a bytes.Buffer first; only on success set Content-Type/WriteHeader and copy the buffer to w. On failure write a clean 500 before anything has been sent.
- **[P2] Untrusted titles and names are written to the terminal without control-character sanitization** — `ui/views/views.go:92`
  - Impact: Titles, project names, and memory/strategy fields originate from the control plane and are emitted verbatim into TUI and plain output (projectsLines line 85, graphLines line 115, memoryLines, reportsLines all do the same). A value containing ANSI escape sequences (e.g. "\x1b[2J", "\x1b[31m") can clear or repaint the terminal, and embedded newlines can forge extra nav/status lines or break the fixed-width column layout. The test TestRunUntilEOF only asserts the renderer itself adds no escapes; it does not cover escapes present in data. The HTML surface is safe because html/template escapes, but the TUI is not.
  - Recommendation: Add a sanitizer in the views package that strips or escapes C0/C1 control characters and newlines from all data-derived fields before formatting, and apply it in every *Lines helper. Add a unit test that renders a title containing "\x1b" and a newline and asserts the output contains neither.
- **[P2] Graph view returns early when there is no work, hiding assignments and orphan nodes** — `ui/views/views.go:99`
  - Impact: The ASSIGNMENTS section (line 117) is only reachable when m.Work is non-empty. A project with zero work packages but active or awaiting-approval assignments renders a graph view that claims nothing is there, so the operator loses visibility of in-flight work. Relatedly, graph nodes whose WorkPackageID has no matching entry in m.Work are never rendered at all, because iteration is driven by m.Work rather than the union of both lists.
  - Recommendation: Do not return early; emit the "(no work packages)" line and continue to the ASSIGNMENTS section. Iterate over graph nodes (or the union of nodes and work packages) so nodes without a matching work package are still listed.
- **[P3] ContractVersion is an unbranded string disconnected from CONTRACT_VERSION** — `contracts/typescript/v2/index.ts:7`
  - Impact: 'Envelope.contractVersion' accepts any string, while the module exports 'CONTRACT_VERSION = "2.0"'. The constant is not reflected in the type, so unsupported versions are accepted at compile time and the constant and type can drift independently.
  - Recommendation: Tie the type to the exported constant, for example 'export type ContractVersion = typeof CONTRACT_VERSION;', or use an explicit literal union. If future versions must be accepted, define a separate runtime compatibility check.
- **[P3] TUI tests do not cover cancellation, command errors, or surface errors** — `ui/surfaces/tui/tui_test.go:27`
  - Impact: Every Handle call is made with context.Background() and only the happy path is asserted; nothing rejects a pre-cancelled context, a Refresh failure, or an unknown command (e.g. ":bogus"), so error rendering through View()/Message is unverified. The idempotency check compares only len(s.Audit()) after re-submitting the same string, which also passes if the second submit failed for an unrelated reason (error path, no-op due to state), so it does not actually prove duplicate-key suppression.
  - Recommendation: Add a case using a cancelled context and one using a failing client stub, assert the rendered "error:" message, and strengthen the idempotency assertion by checking the audit entries are unchanged element-wise (or that the same work package ID is still present) rather than only the slice length.
- **[P3] Root route pattern matches every GET path** — `ui/surfaces/web/web.go:31`
  - Impact: In the Go 1.22 mux, the pattern "GET /" is a subtree match, so requests to any path (e.g. /favicon.ico, /admin, /does-not-exist) return 200 with the dashboard instead of 404. This makes probing and log analysis misleading and hides typos in links.
  - Recommendation: Register the exact page route as "GET /{$}" (Go 1.22 end-of-path anchor) and let unmatched paths fall through to the default 404.
- **[P3] Internal error strings are echoed to HTTP clients** — `ui/surfaces/web/web.go:43`
  - Impact: state.Load and template execution errors are returned verbatim in the response body (lines 43 and 48). Backend errors from the control plane can carry internal paths, hostnames, RPC details, or project identifiers that a local web client should not see.
  - Recommendation: Log the full error server-side and return a fixed message such as "dashboard unavailable" / "render failed" to the client.
- **[P3] Web test ignores read errors and never exercises the read-only or failure guarantees** — `ui/surfaces/web/web_test.go:22`
  - Impact: ReadAll errors are discarded and the /plain response status is never checked, so a 500 or truncated body could pass. More importantly, TestWebReadOnly asserts nothing about the read-only invariant it is named after: no POST/PUT/DELETE is attempted to confirm the handler has no command path, and the 502 branch in page/plain is untested, so the error handling added for state.Load failures has no coverage.
  - Recommendation: Check resp.StatusCode and ReadAll errors for both endpoints, add a non-GET request asserting 405/404, and add a test with a client stub whose CreateProject/fetch calls fail to assert the 502 path and that no internal error text is leaked.
- **[P3] Report section is suppressed when the session list is empty** — `ui/views/views.go:129`
  - Impact: When m.Report.SessionID is populated but m.Sessions is empty, the report detail (events, sequence range, source/confidence breakdown, work packages, actors) is silently dropped, even though the web page renders .Report independently of .Sessions. Users get an empty REPORTS view in one surface and a populated one in the other for the same state.
  - Recommendation: Guard the sessions list separately from the report block: append "(no sessions)" when empty and still render the REPORT section when Report.SessionID is set.

### Validation

- findings: P2=4, P3=6
- block threshold: P1
- coverage: 14/75 files (7 batches failed) — INCOMPLETE
- delta: new=10 ongoing=0 regressed=0 resolved=0

### Risk

Medium

### Recommendation

Needs Investigation
