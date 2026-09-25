// Package policies is the durable policy-definition store.
//
// A policy definition catalogues what policies exist (for example approval,
// budget, routing, and gate policies). Which policies apply to a particular
// workflow, project, or execution is decided elsewhere; applicability is not a
// registry concern. The specification types land with contracts v2; this
// package is a reserved architectural boundary.
package policies
