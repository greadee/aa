package archtest

import "path/filepath"

// Modules returns the aa product modules in dependency order.
func Modules(root string) []Module {
	names := []string{
		"contracts", "registry", "obsv", "memory", "sync", "toolbox",
		"forge", "runtime", "kernel", "visualizer", "ui",
	}
	mods := make([]Module, 0, len(names))
	for _, name := range names {
		mods = append(mods, Module{Name: name, Dir: filepath.Join(root, name)})
	}
	return mods
}

// Allowed returns the permitted direct-and-transitive aa dependencies per module.
//
//	contracts depends on nothing. A module may depend only on modules below it:
//
//	registry   -> contracts
//	obsv       -> contracts
//	memory     -> contracts, obsv
//	sync       -> contracts
//	toolbox    -> contracts
//	forge      -> contracts, toolbox
//	runtime    -> contracts, registry
//	kernel     -> contracts, registry, obsv, memory, toolbox, runtime
//	visualizer -> contracts, obsv, memory
//	ui         -> contracts
//
// kernel reaches sync, forge, and sifter over RPC, not by import.
func Allowed() map[string]map[string]bool {
	set := func(names ...string) map[string]bool {
		m := make(map[string]bool, len(names))
		for _, n := range names {
			m[n] = true
		}
		return m
	}
	return map[string]map[string]bool{
		"contracts":  set(),
		"registry":   set("contracts"),
		"obsv":       set("contracts"),
		"memory":     set("contracts", "obsv"),
		"sync":       set("contracts"),
		"toolbox":    set("contracts"),
		"forge":      set("contracts", "toolbox"),
		"runtime":    set("contracts", "registry"),
		"kernel":     set("contracts", "registry", "obsv", "memory", "toolbox", "runtime"),
		"visualizer": set("contracts", "obsv", "memory"),
		"ui":         set("contracts"),
	}
}

// CheckWorkspace runs the layering check across all product modules.
func CheckWorkspace(root string) ([]Violation, error) {
	allowed := Allowed()
	var violations []Violation
	for _, mod := range Modules(root) {
		deps, err := ListDeps(mod.Dir)
		if err != nil {
			return nil, err
		}
		violations = append(violations, Check(mod.Name, allowed[mod.Name], deps)...)
	}
	return violations, nil
}
