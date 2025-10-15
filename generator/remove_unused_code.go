package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
)

// RemoveUnusedCode aggressively removes top-level types, functions, and methods
// that are not reachable starting from roots: main + any top-level var/function
// assignments to function identifiers (e.g., var solveX = solveC). This is a
// heuristic, sufficient for competitive programming consolidation to reduce
// binary size. It does NOT perform full type-resolution; it relies on textual
// /AST patterns and may keep some extra code in ambiguous cases, but aims never
// to remove actually used code.
func RemoveUnusedCode(src []byte) []byte {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "merged.go", src, parser.ParseComments)
	if err != nil {
		return src // fallback on parse failure
	}

	// Collect declarations
	funcDecls := map[string]*ast.FuncDecl{}
	// Methods mapped by method name -> list (receiver-insensitive; conservative)
	methodDecls := map[string][]*ast.FuncDecl{}
	// Type declarations by name
	typeDecls := map[string]*ast.TypeSpec{}

	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Recv == nil {
				funcDecls[fd.Name.Name] = fd
			} else {
				methodDecls[fd.Name.Name] = append(methodDecls[fd.Name.Name], fd)
			}
			continue
		}
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			for _, sp := range gd.Specs {
				if ts, ok2 := sp.(*ast.TypeSpec); ok2 {
					typeDecls[ts.Name.Name] = ts
				}
			}
		}
	}

	// Root functions: main + functions referenced by top-level variable initializations.
	rootFuncs := map[string]bool{"main": true}
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || (gd.Tok != token.VAR && gd.Tok != token.CONST) {
			continue
		}
		for _, sp := range gd.Specs {
			vs, ok2 := sp.(*ast.ValueSpec)
			if !ok2 {
				continue
			}
			for _, val := range vs.Values {
				if id, ok3 := val.(*ast.Ident); ok3 {
					if _, exists := funcDecls[id.Name]; exists {
						rootFuncs[id.Name] = true
					}
				}
			}
		}
	}

	// Reachability analysis: BFS over functions & methods.
	reachableFuncs := map[string]bool{}
	reachableMethods := map[string]bool{}
	reachableTypes := map[string]bool{}

	queue := []string{}
	for f := range rootFuncs {
		if _, ok := funcDecls[f]; ok {
			queue = append(queue, f)
		}
	}

	processFuncBody := func(fd *ast.FuncDecl) {
		if fd == nil || fd.Body == nil {
			return
		}
		// Collect parameter & result types as used types
		collectTypeExpr := func(e ast.Expr) {
			for _, name := range extractTypeIdents(e) {
				reachableTypes[name] = true
			}
		}
		if fd.Type != nil {
			if fd.Type.Params != nil {
				for _, f := range fd.Type.Params.List {
					collectTypeExpr(f.Type)
				}
			}
			if fd.Type.Results != nil {
				for _, f := range fd.Type.Results.List {
					collectTypeExpr(f.Type)
				}
			}
		}
		// Receiver type
		if fd.Recv != nil {
			for _, r := range fd.Recv.List {
				for _, name := range extractTypeIdents(r.Type) {
					reachableTypes[name] = true
				}
			}
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				// function call
				switch fun := x.Fun.(type) {
				case *ast.Ident:
					if _, isFn := funcDecls[fun.Name]; isFn {
						if !reachableFuncs[fun.Name] {
							queue = append(queue, fun.Name)
						}
					}
				case *ast.SelectorExpr:
					// method call; track selector name
					reachableMethods[fun.Sel.Name] = true
				}
			case *ast.CompositeLit:
				for _, name := range extractTypeIdents(x.Type) {
					reachableTypes[name] = true
				}
			case *ast.TypeAssertExpr:
				for _, name := range extractTypeIdents(x.Type) {
					reachableTypes[name] = true
				}
			case *ast.UnaryExpr:
				if se, ok := x.X.(*ast.CompositeLit); ok {
					for _, name := range extractTypeIdents(se.Type) {
						reachableTypes[name] = true
					}
				}
			case *ast.ValueSpec:
				if x.Type != nil {
					for _, name := range extractTypeIdents(x.Type) {
						reachableTypes[name] = true
					}
				}
			}
			return true
		})
	}

	for len(queue) > 0 {
		fnName := queue[0]
		queue = queue[1:]
		if reachableFuncs[fnName] {
			continue
		}
		reachableFuncs[fnName] = true
		processFuncBody(funcDecls[fnName])
	}

	// Iterative BFS over methods: newly discovered method calls inside methods enqueue further processing.
	processedMethods := map[string]bool{}
	methodQueue := []string{}
	for mName := range reachableMethods { // seed queue with methods found while scanning root functions
		methodQueue = append(methodQueue, mName)
	}
	for len(methodQueue) > 0 {
		mName := methodQueue[0]
		methodQueue = methodQueue[1:]
		if processedMethods[mName] { // already expanded
			continue
		}
		processedMethods[mName] = true
		decls := methodDecls[mName]
		for _, md := range decls {
			processFuncBody(md)
		}
		// Enqueue any newly discovered methods not yet processed
		for newName := range reachableMethods {
			if !processedMethods[newName] {
				// ensure it actually exists as a declaration
				if len(methodDecls[newName]) > 0 {
					alreadyInQueue := false
					for _, qn := range methodQueue {
						if qn == newName {
							alreadyInQueue = true
							break
						}
					}
					if !alreadyInQueue {
						methodQueue = append(methodQueue, newName)
					}
				}
			}
		}
	}

	// Second BFS pass for functions discovered while expanding methods.
	for len(queue) > 0 {
		fnName := queue[0]
		queue = queue[1:]
		if reachableFuncs[fnName] {
			continue
		}
		reachableFuncs[fnName] = true
		processFuncBody(funcDecls[fnName])
		// Newly found methods during function expansion should also be BFS processed.
		for newName := range reachableMethods {
			if !processedMethods[newName] && len(methodDecls[newName]) > 0 {
				methodQueue = append(methodQueue, newName)
			}
		}
		// Drain any methodQueue additions created here
		for len(methodQueue) > 0 {
			mName := methodQueue[0]
			methodQueue = methodQueue[1:]
			if processedMethods[mName] {
				continue
			}
			processedMethods[mName] = true
			for _, md := range methodDecls[mName] {
				processFuncBody(md)
			}
			for newName := range reachableMethods {
				if !processedMethods[newName] && len(methodDecls[newName]) > 0 {
					methodQueue = append(methodQueue, newName)
				}
			}
		}
	}

	// Ensure types used by reachable method receivers retained.
	for _, list := range methodDecls {
		for _, md := range list {
			if reachableMethods[md.Name.Name] && md.Recv != nil {
				for _, r := range md.Recv.List {
					for _, name := range extractTypeIdents(r.Type) {
						reachableTypes[name] = true
					}
				}
			}
		}
	}

	// Build new decl list filtering out unused ones.
	var newDecls []ast.Decl
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Recv == nil { // normal function
				if fd.Name.Name == "main" || reachableFuncs[fd.Name.Name] {
					newDecls = append(newDecls, d)
				}
			} else { // method
				if reachableMethods[fd.Name.Name] {
					newDecls = append(newDecls, d)
				}
			}
			continue
		}
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			// Keep only used type specs inside this GenDecl
			var keptSpecs []ast.Spec
			for _, sp := range gd.Specs {
				if ts, ok2 := sp.(*ast.TypeSpec); ok2 {
					if reachableTypes[ts.Name.Name] {
						keptSpecs = append(keptSpecs, sp)
					}
				}
			}
			if len(keptSpecs) > 0 {
				gd.Specs = keptSpecs
				newDecls = append(newDecls, gd)
			}
			continue
		}
		// Retain non-type declarations (vars/consts) as they may influence logic or root detection; optional pruning could be added.
		newDecls = append(newDecls, d)
	}
	file.Decls = newDecls

	// Remove all comments entirely (both line and block) per request.
	file.Comments = nil
	// Also clear Doc comments attached to declarations for safety.
	for _, d := range file.Decls {
		if gd, ok := d.(*ast.GenDecl); ok {
			gd.Doc = nil
		}
		if fd, ok := d.(*ast.FuncDecl); ok {
			fd.Doc = nil
		}
	}

	var out bytes.Buffer
	_ = printer.Fprint(&out, fset, file)
	return out.Bytes()
}

// extractTypeIdents returns identifier names referenced in a type expression.
func extractTypeIdents(e ast.Expr) []string {
	var names []string
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			// Exclude blank identifier
			if id.Name != "_" && id.Name != "" && !isBuiltin(id.Name) {
				names = append(names, id.Name)
			}
		}
		return true
	})
	return names
}

// Minimal builtin exclusion to avoid keeping pseudo types like 'int'.
func isBuiltin(name string) bool {
	// A tiny set sufficient for our pruning context.
	switch name {
	case "int", "uint", "uint64", "uint32", "byte", "rune", "string", "bool", "float64", "complex128", "error":
		return true
	}
	return false
}

// (Optional) helper to debug sets; unused but handy for future tweaks.
