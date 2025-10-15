package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
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
	// Global value/const declarations by name (single identifier specs only)
	valueDecls := map[string]*ast.ValueSpec{}

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
			continue
		}
		if gd, ok := d.(*ast.GenDecl); ok && (gd.Tok == token.CONST || gd.Tok == token.VAR) {
			for _, sp := range gd.Specs {
				if vs, ok2 := sp.(*ast.ValueSpec); ok2 {
					for _, nm := range vs.Names {
						valueDecls[nm.Name] = vs
					}
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
	reachableValues := map[string]bool{}

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
			case *ast.Ident:
				// Track identifier usage for global value/const declarations
				if _, ok := valueDecls[x.Name]; ok {
					reachableValues[x.Name] = true
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
		if gd, ok := d.(*ast.GenDecl); ok && (gd.Tok == token.CONST || gd.Tok == token.VAR) {
			var keptSpecs []ast.Spec
			for _, sp := range gd.Specs {
				if vs, ok2 := sp.(*ast.ValueSpec); ok2 {
					keep := false
					for _, nm := range vs.Names {
						if reachableValues[nm.Name] {
							keep = true
							break
						}
					}
					if keep {
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

	file.Decls = SortDeclarations(file.Decls)

	ordered := buildPreorderSource(file, fset)
	ordered = collapseBlockBlankLines(ordered, "const (")
	ordered = collapseBlockBlankLines(ordered, "var (")
	ordered = ensureFunctionGaps(ordered)
	return []byte(ordered)
}

// END RemoveUnusedCode

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

// collapseBlockBlankLines removes extra blank lines inside a parenthesized decl block.
func collapseBlockBlankLines(src, prefix string) string {
	idx := strings.Index(src, prefix)
	if idx == -1 {
		return src
	}
	start := idx + len(prefix)
	depth := 1
	for i := start; i < len(src); i++ {
		c := src[i]
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth == 0 {
				block := src[idx : i+1]
				lines := strings.Split(block, "\n")
				var cleaned []string
				for _, ln := range lines {
					if strings.TrimSpace(ln) == "" { // drop all blank lines inside block
						continue
					}
					cleaned = append(cleaned, ln)
				}
				newBlock := strings.Join(cleaned, "\n")
				return src[:idx] + newBlock + src[i+1:]
			}
		}
	}
	return src
}

// buildPreorderSource orders declarations in a pre-order traversal starting from main
// following first usage of functions, methods, types, vars, consts as they appear.
func buildPreorderSource(file *ast.File, fset *token.FileSet) string {
	// Collect declarations
	funcDecls := map[string]*ast.FuncDecl{}
	methodDecls := map[string][]*ast.FuncDecl{}
	typeDecls := map[string]*ast.TypeSpec{}
	valueDecls := map[string]struct {
		vs  *ast.ValueSpec
		tok token.Token
	}{}
	var mainDecl *ast.FuncDecl
	var imports []*ast.GenDecl
	for _, d := range file.Decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			if x.Recv == nil {
				if x.Name.Name == "main" {
					mainDecl = x
				}
				funcDecls[x.Name.Name] = x
			} else {
				methodDecls[x.Name.Name] = append(methodDecls[x.Name.Name], x)
			}
		case *ast.GenDecl:
			if x.Tok == token.IMPORT {
				imports = append(imports, x)
			}
			if x.Tok == token.TYPE {
				for _, sp := range x.Specs {
					if ts, ok := sp.(*ast.TypeSpec); ok {
						typeDecls[ts.Name.Name] = ts
					}
				}
			}
			if x.Tok == token.CONST || x.Tok == token.VAR {
				for _, sp := range x.Specs {
					if vs, ok := sp.(*ast.ValueSpec); ok {
						for _, nm := range vs.Names {
							valueDecls[nm.Name] = struct {
								vs  *ast.ValueSpec
								tok token.Token
							}{vs, x.Tok}
						}
					}
				}
			}
		}
	}

	visited := map[string]bool{}
	order := []string{}
	stack := []string{}
	if mainDecl != nil {
		stack = append(stack, "main")
	}

	addName := func(name string) {
		if name == "main" { // already handled explicitly
			return
		}
		if visited[name] {
			return
		}
		visited[name] = true
		order = append(order, name)
		// Depth-first traversal: push onto stack for further expansion if it has a body
		if funcDecls[name] != nil || len(methodDecls[name]) > 0 {
			stack = append(stack, name)
		}
	}

	extractUsed := func(fd *ast.FuncDecl) []string {
		used := []string{}
		if fd == nil || fd.Body == nil {
			return used
		}
		seenLocal := map[string]bool{}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				switch fun := x.Fun.(type) {
				case *ast.Ident:
					if funcDecls[fun.Name] != nil && !seenLocal[fun.Name] {
						used = append(used, fun.Name)
						seenLocal[fun.Name] = true
					}
				case *ast.SelectorExpr:
					if methodDecls[fun.Sel.Name] != nil && !seenLocal[fun.Sel.Name] {
						used = append(used, fun.Sel.Name)
						seenLocal[fun.Sel.Name] = true
					}
				}
			case *ast.CompositeLit:
				for _, tn := range extractTypeIdents(x.Type) {
					if typeDecls[tn] != nil && !seenLocal[tn] {
						used = append(used, tn)
						seenLocal[tn] = true
					}
				}
			case *ast.TypeAssertExpr:
				for _, tn := range extractTypeIdents(x.Type) {
					if typeDecls[tn] != nil && !seenLocal[tn] {
						used = append(used, tn)
						seenLocal[tn] = true
					}
				}
			case *ast.UnaryExpr:
				if cl, ok := x.X.(*ast.CompositeLit); ok {
					for _, tn := range extractTypeIdents(cl.Type) {
						if typeDecls[tn] != nil && !seenLocal[tn] {
							used = append(used, tn)
							seenLocal[tn] = true
						}
					}
				}
			case *ast.Ident:
				if valueDecls[x.Name].vs != nil && !seenLocal[x.Name] {
					used = append(used, x.Name)
					seenLocal[x.Name] = true
				}
			case *ast.ValueSpec:
				if x.Type != nil {
					for _, tn := range extractTypeIdents(x.Type) {
						if typeDecls[tn] != nil && !seenLocal[tn] {
							used = append(used, tn)
							seenLocal[tn] = true
						}
					}
				}
			}
			return true
		})
		return used
	}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == "main" {
			if !visited[cur] {
				visited[cur] = true
				// main at top, but we don't add to order list (printed separately)
				for _, u := range extractUsed(mainDecl) {
					addName(u)
				}
			}
			continue
		}
		// Functions
		if fd := funcDecls[cur]; fd != nil {
			for _, u := range extractUsed(fd) {
				addName(u)
			}
		}
		// Methods (multiple with same name)
		if mds := methodDecls[cur]; len(mds) > 0 {
			for _, md := range mds {
				for _, u := range extractUsed(md) {
					addName(u)
				}
			}
		}
	}

	// Build output
	printNode := func(n ast.Node) string { var b bytes.Buffer; _ = printer.Fprint(&b, fset, n); return b.String() }

	type emission struct {
		kind token.Token // TYPE, CONST, VAR or ILLEGAL for funcs/methods/package/import sentinel
		spec ast.Spec    // for TYPE/CONST/VAR single specs
		node ast.Node    // for funcs/methods/import blocks
	}
	var emits []emission

	var packageHeader bytes.Buffer
	packageHeader.WriteString("package " + file.Name.Name + "\n\n")
	for _, im := range imports {
		packageHeader.WriteString(printNode(im) + "\n")
	}
	if len(imports) > 0 {
		packageHeader.WriteString("\n")
	}
	if mainDecl != nil {
		packageHeader.WriteString(printNode(mainDecl) + "\n\n")
	}

	printed := map[string]bool{"main": true}
	emittedValueSpec := map[*ast.ValueSpec]bool{}

	// helper to add type spec ensuring no duplicate
	addTypeSpec := func(ts *ast.TypeSpec) { emits = append(emits, emission{kind: token.TYPE, spec: ts}) }
	addValueSpec := func(tok token.Token, vs *ast.ValueSpec) { emits = append(emits, emission{kind: tok, spec: vs}) }
	addFunc := func(fd *ast.FuncDecl) { emits = append(emits, emission{kind: token.ILLEGAL, node: fd}) }

	for _, name := range order {
		if printed[name] {
			continue
		}
		printed[name] = true
		if ts := typeDecls[name]; ts != nil {
			addTypeSpec(ts)
			continue
		}
		if info, ok := valueDecls[name]; ok && info.vs != nil {
			if !emittedValueSpec[info.vs] {
				addValueSpec(info.tok, info.vs)
				emittedValueSpec[info.vs] = true
			}
			continue
		}
		if fd := funcDecls[name]; fd != nil {
			addFunc(fd)
			continue
		}
		if mds := methodDecls[name]; len(mds) > 0 {
			for _, md := range mds {
				// receiver type first if not printed
				if md.Recv != nil && len(md.Recv.List) > 0 {
					recvType := md.Recv.List[0].Type
					if se, ok := recvType.(*ast.StarExpr); ok {
						recvType = se.X
					}
					if id, ok := recvType.(*ast.Ident); ok {
						if !printed[id.Name] {
							if ts := typeDecls[id.Name]; ts != nil {
								addTypeSpec(ts)
								printed[id.Name] = true
							}
						}
					}
				}
				addFunc(md)
			}
			continue
		}
	}

	appendDecl := func(name string) {
		if printed[name] {
			return
		}
		printed[name] = true
		if ts := typeDecls[name]; ts != nil {
			addTypeSpec(ts)
			return
		}
		if info, ok := valueDecls[name]; ok && info.vs != nil {
			if !emittedValueSpec[info.vs] {
				addValueSpec(info.tok, info.vs)
				emittedValueSpec[info.vs] = true
			}
			return
		}
		if fd := funcDecls[name]; fd != nil {
			addFunc(fd)
			return
		}
		if mds := methodDecls[name]; len(mds) > 0 {
			for _, md := range mds {
				if md.Recv != nil && len(md.Recv.List) > 0 {
					recvType := md.Recv.List[0].Type
					if se, ok := recvType.(*ast.StarExpr); ok {
						recvType = se.X
					}
					if id, ok := recvType.(*ast.Ident); ok {
						if !printed[id.Name] {
							if ts := typeDecls[id.Name]; ts != nil {
								addTypeSpec(ts)
								printed[id.Name] = true
							}
						}
					}
				}
				addFunc(md)
			}
			return
		}
	}

	for name := range typeDecls {
		appendDecl(name)
	}
	for name := range valueDecls {
		appendDecl(name)
	}
	for name := range funcDecls {
		appendDecl(name)
	}
	for name := range methodDecls {
		appendDecl(name)
	}

	// Group consecutive specs of same kind (TYPE, CONST, VAR)
	var out bytes.Buffer
	out.WriteString(packageHeader.String())
	i := 0
	for i < len(emits) {
		// groupable tokens
		if emits[i].kind == token.TYPE || emits[i].kind == token.CONST || emits[i].kind == token.VAR {
			kind := emits[i].kind
			j := i
			var specs []ast.Spec
			for j < len(emits) && emits[j].kind == kind {
				if emits[j].spec != nil {
					specs = append(specs, emits[j].spec)
					j++
					continue
				}
				break // encounter non-spec emission, stop grouping
			}
			if len(specs) > 1 { // create multi-spec block
				gd := &ast.GenDecl{Tok: kind, Specs: specs}
				out.WriteString(printNode(gd) + "\n\n")
				i = j
				continue
			} else if len(specs) == 1 { // single spec
				gd := &ast.GenDecl{Tok: kind, Specs: specs}
				out.WriteString(printNode(gd) + "\n\n")
				i = j
				continue
			}
		}
		// non-groupable (func/method/import etc.)
		if emits[i].node != nil {
			out.WriteString(printNode(emits[i].node) + "\n\n")
		}
		i++
	}

	return out.String()
}

// ensureFunctionGaps inserts exactly one blank line before each top-level func/method
// declaration (except the first function in the file) and removes duplicate blank lines
// directly preceding functions.
func ensureFunctionGaps(src string) string {
	lines := strings.Split(src, "\n")
	var out []string
	funcCount := 0
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		isFunc := strings.HasPrefix(trim, "func ")
		if isFunc {
			// backtrack over trailing blank lines already emitted
			for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
				out = out[:len(out)-1]
			}
			if funcCount > 0 { // not first function
				out = append(out, "")
			}
			funcCount++
		}
		out = append(out, ln)
		// no-op: we removed prevNonEmptyIndex tracking
	}
	return strings.Join(out, "\n")
}
