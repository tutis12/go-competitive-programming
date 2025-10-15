package main

import (
	"go/ast"
	"go/token"
	"sort"
)

// SortDeclarations lexicographically orders global declarations (types, vars, consts, funcs, methods)
// keeping import declarations at the top in original order. Multi-spec type/value decls are split
// into single-spec decls for stable ordering without altering semantics.
func SortDeclarations(decls []ast.Decl) []ast.Decl {
	var importDecls []ast.Decl
	// Grouped by kind: const, var, type, functions (no receiver), methods (have receiver), other.
	type entry struct {
		name string
		decl ast.Decl
	}
	var consts, vars, types, functions, methods, others []entry

	for _, d := range decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			importDecls = append(importDecls, d)
			continue
		}
		if gd, ok := d.(*ast.GenDecl); ok && (gd.Tok == token.CONST || gd.Tok == token.VAR || gd.Tok == token.TYPE) {
			for _, sp := range gd.Specs {
				switch spec := sp.(type) {
				case *ast.TypeSpec:
					newDecl := &ast.GenDecl{Tok: gd.Tok, Specs: []ast.Spec{spec}}
					entryItem := entry{spec.Name.Name, newDecl}
					if gd.Tok == token.TYPE {
						types = append(types, entryItem)
					} else if gd.Tok == token.CONST {
						consts = append(consts, entryItem)
					} else {
						vars = append(vars, entryItem)
					}
				case *ast.ValueSpec:
					if len(spec.Names) > 0 {
						name := spec.Names[0].Name
						newDecl := &ast.GenDecl{Tok: gd.Tok, Specs: []ast.Spec{spec}}
						entryItem := entry{name, newDecl}
						if gd.Tok == token.CONST {
							consts = append(consts, entryItem)
						} else {
							vars = append(vars, entryItem)
						}
					}
				}
			}
			continue
		}
		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Recv == nil {
				functions = append(functions, entry{fd.Name.Name, d})
			} else {
				methods = append(methods, entry{fd.Name.Name, d})
			}
			continue
		}
		others = append(others, entry{"", d})
	}
	// Sort each group lexicographically by name (stable for identical names).
	sort.SliceStable(consts, func(i, j int) bool { return consts[i].name < consts[j].name })
	sort.SliceStable(vars, func(i, j int) bool { return vars[i].name < vars[j].name })
	sort.SliceStable(types, func(i, j int) bool { return types[i].name < types[j].name })
	sort.SliceStable(functions, func(i, j int) bool { return functions[i].name < functions[j].name })
	sort.SliceStable(methods, func(i, j int) bool { return methods[i].name < methods[j].name })
	// Others not sorted (could sort if name non-empty).

	var reordered []ast.Decl
	reordered = append(reordered, importDecls...)
	// Aggregate consts into single const() block, vars into single var() block.
	if len(consts) > 0 {
		agg := &ast.GenDecl{Tok: token.CONST}
		for _, c := range consts {
			// Each c.decl is a GenDecl with single spec
			if gd, ok := c.decl.(*ast.GenDecl); ok {
				agg.Specs = append(agg.Specs, gd.Specs...)
			}
		}
		reordered = append(reordered, agg)
	}
	if len(vars) > 0 {
		agg := &ast.GenDecl{Tok: token.VAR}
		for _, v := range vars {
			if gd, ok := v.decl.(*ast.GenDecl); ok {
				agg.Specs = append(agg.Specs, gd.Specs...)
			}
		}
		reordered = append(reordered, agg)
	}
	for _, t := range types { // keep types as individual for clarity
		reordered = append(reordered, t.decl)
	}
	for _, f := range functions {
		reordered = append(reordered, f.decl)
	}
	for _, m := range methods {
		reordered = append(reordered, m.decl)
	}
	for _, o := range others {
		reordered = append(reordered, o.decl)
	}
	return reordered
}
