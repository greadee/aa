// Package routines is the durable routine-definition store.
//
// A routine is an organization-owned, reusable specification for recurring
// work. A project deploys a routine as a workflow; the workflow engine lives in
// toolbox, and the routine definition lives here. Registry stores and serves
// contracts v2 Routine definitions; it owns no workflow runtime.
package routines
