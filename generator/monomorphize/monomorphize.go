package monomorphize

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

// Shared key type for instantiation maps (avoid anonymous-struct mismatches).
type instKey struct {
	Name string
	Args string
}

// record for an instantiation we discovered
type instRecord struct {
	args     []ast.Expr   // cloned exprs used for naming/substitution
	argTypes []types.Type // types for hoistability checks (nil => unknown => non-hoistable)
}

// --- DROP-IN REPLACEMENT: Monomorphize with debug formatting on failure ---
func Monomorphize(src []byte) []byte {
	const maxIters = 2000 // hard stop to avoid accidental infinite loops

	// We keep track of which instantiations we already created across iterations.
	// Persisting this map avoids re-creating the same concrete decl when we reparse.
	created := map[instKey]string{}

	for iter := 0; iter < maxIters; iter++ {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "input.go", src, parser.ParseComments)
		if err != nil {
			panic(err)
		}

		// Index generic decls / methods
		genTypes := map[string]*ast.TypeSpec{}
		genFuncs := map[string]*ast.FuncDecl{}
		methodsByRecv := map[string][]*ast.FuncDecl{}
		for _, d := range file.Decls {
			switch dd := d.(type) {
			case *ast.GenDecl:
				if dd.Tok == token.TYPE {
					for _, spec := range dd.Specs {
						ts := spec.(*ast.TypeSpec)
						if ts.TypeParams != nil && len(ts.TypeParams.List) > 0 {
							genTypes[ts.Name.Name] = ts
						}
					}
				}
			case *ast.FuncDecl:
				if dd.Type != nil && dd.Type.TypeParams != nil && len(dd.Type.TypeParams.List) > 0 && dd.Name != nil {
					genFuncs[dd.Name.Name] = dd
				}
				if dd.Recv != nil && len(dd.Recv.List) == 1 {
					base, _ := baseIdentOfReceiver(dd.Recv.List[0].Type)
					if base != "" {
						if _, ok := genTypes[base]; ok {
							methodsByRecv[base] = append(methodsByRecv[base], dd)
						}
					}
				}
			}
		}

		// Type-check current file (must compile every iteration).
		info := &types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Instances:  make(map[*ast.Ident]types.Instance),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Implicits:  make(map[ast.Node]types.Object),
		}
		conf := &types.Config{
			Importer: importer.Default(),
		}
		// Always print a highlighted snippet if go/types emits an error via callback.
		conf.Error = func(e error) {
			debugTypecheckFailure(fset, file, e)
			panic(e)
		}
		// And also if Check returns an error.
		if _, err := conf.Check(file.Name.Name, fset, []*ast.File{file}, info); err != nil {
			debugTypecheckFailure(fset, file, err)
			panic(err)
		}

		// ---- Hoist local (block-scoped) types to package scope first ----
		if hoistLocalTypes(file, fset, info) {
			// If formatting fails, we print the offending line/col inside this helper.
			src = formatToBytesOrDie(fset, file)
			// restart iteration with fresh parse/type info
			continue
		}

		// Discover instantiations in this iteration
		insts := map[instKey]instRecord{}
		callKeys := map[*ast.CallExpr]instKey{}
		collect := func(n ast.Node) {
			ast.Inspect(n, func(n ast.Node) bool {
				switch ix := n.(type) {
				case *ast.IndexListExpr:
					var baseName string
					switch b := ix.X.(type) {
					case *ast.Ident:
						baseName = b.Name
					case *ast.SelectorExpr:
						baseName = b.Sel.Name
					}
					if baseName != "" && (genTypes[baseName] != nil || genFuncs[baseName] != nil) {
						k := instKey{Name: baseName, Args: argsKey(ix.Indices)}
						var ts []types.Type
						for _, a := range ix.Indices {
							if tv, ok := info.Types[a]; ok && tv.Type != nil {
								ts = append(ts, tv.Type)
								continue
							}
							if id, ok := a.(*ast.Ident); ok {
								if pt := predeclTypeByName(id.Name); pt != nil {
									ts = append(ts, pt)
									continue
								}
							}
							ts = append(ts, nil)
						}
						insts[k] = instRecord{args: cloneExprList(ix.Indices), argTypes: ts}
					}
				case *ast.IndexExpr:
					var baseName string
					switch b := ix.X.(type) {
					case *ast.Ident:
						baseName = b.Name
					case *ast.SelectorExpr:
						baseName = b.Sel.Name
					}
					if baseName != "" && (genTypes[baseName] != nil || genFuncs[baseName] != nil) {
						k := instKey{Name: baseName, Args: argsKey([]ast.Expr{ix.Index})}
						var targ types.Type
						if tv, ok := info.Types[ix.Index]; ok && tv.Type != nil {
							targ = tv.Type
						} else if id, ok := ix.Index.(*ast.Ident); ok {
							if pt := predeclTypeByName(id.Name); pt != nil {
								targ = pt
							}
						}
						insts[k] = instRecord{args: []ast.Expr{cloneExpr(ix.Index)}, argTypes: []types.Type{targ}}
					}
				case *ast.CallExpr:
					if id, ok := ix.Fun.(*ast.Ident); ok {
						if inst, ok := info.Instances[id]; ok && inst.TypeArgs.Len() > 0 {
							var baseName string
							if obj := info.Uses[id]; obj != nil {
								baseName = obj.Name()
							} else {
								baseName = id.Name
							}
							if _, isGen := genFuncs[baseName]; isGen {
								var exprArgs []ast.Expr
								var typeArgs []types.Type
								for i := 0; i < inst.TypeArgs.Len(); i++ {
									ta := inst.TypeArgs.At(i)
									typeArgs = append(typeArgs, ta)
									exprArgs = append(exprArgs, typeToExprWithFile(file, ta))
								}
								k := instKey{Name: baseName, Args: argsKey(exprArgs)}
								insts[k] = instRecord{args: cloneExprList(exprArgs), argTypes: typeArgs}
								callKeys[ix] = k
							}
						}
					}
				}
				return true
			})
		}
		collect(file)

		// Build a deterministic list, then pick ONE instantiation to materialize.
		type instEntry struct {
			key      instKey
			args     []ast.Expr
			argTypes []types.Type
		}
		entries := make([]instEntry, 0, len(insts))
		for k, rec := range insts {
			entries = append(entries, instEntry{key: k, args: rec.args, argTypes: rec.argTypes})
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].key.Name != entries[j].key.Name {
				return entries[i].key.Name < entries[j].key.Name
			}
			return entries[i].key.Args < entries[j].key.Args
		})

		madeChange := false

		// Choose the first hoistable, not-yet-created instantiation
		for _, e := range entries {
			k, args, argTypes := e.key, e.args, e.argTypes
			if _, done := created[k]; done {
				continue
			}
			// hoistability check
			hoistOK := true
			for _, t := range argTypes {
				if t == nil || !isHoistableType(t) {
					hoistOK = false
					break
				}
			}
			if !hoistOK {
				continue
			}

			concreteName := makeMonoName(k.Name, args)
			created[k] = concreteName

			// Materialize exactly one concrete decl (and its methods if type)
			switch {
			case genTypes[k.Name] != nil:
				ts := genTypes[k.Name]
				params := flattenFieldListIdents(ts.TypeParams)
				if len(params) != len(args) {
					panic(fmt.Errorf("type %s: expected %d type args, got %d", k.Name, len(params), len(args)))
				}
				subst := map[string]ast.Expr{}
				for i, p := range params {
					subst[p] = cloneExpr(args[i])
				}

				newDecl := cloneTypeDecl(ts, concreteName, subst)
				appendDecl(file, newDecl)

				recvMethods := append([]*ast.FuncDecl(nil), methodsByRecv[k.Name]...)
				sort.SliceStable(recvMethods, func(i, j int) bool {
					return fset.Position(recvMethods[i].Pos()).Offset < fset.Position(recvMethods[j].Pos()).Offset
				})
				for _, m := range recvMethods {
					newM := cloneMethodForConcrete(m, k.Name, concreteName, subst)
					appendDecl(file, newM)
				}

			default:
				fd := getFuncDecl(file, genFuncs, k.Name)
				if fd != nil && fd.Type != nil {
					params := flattenFieldListIdents(fd.Type.TypeParams)
					if len(params) != len(args) {
						panic(fmt.Errorf("func %s: expected %d type args, got %d", k.Name, len(params), len(args)))
					}
					subst := map[string]ast.Expr{}
					for i, p := range params {
						subst[p] = cloneExpr(args[i])
					}
					newFunc := cloneFuncForConcrete(fd, concreteName, subst)
					appendDecl(file, newFunc)
				}
			}

			// Rewrite usages for ALL known instantiations so nested ones get updated too
			_ = rewriteInstantiations(file, created, callKeys)

			// Emit fresh source and restart the outer loop with a new parse/typecheck.
			src = formatToBytesOrDie(fset, file)
			madeChange = true
			break // <-- enforce "one replacement per iteration"
		}

		if !madeChange {
			// Nothing left to do in this iteration => done.
			return src
		}
	}

	panic("Monomorphize: exceeded iteration limit; possible cycle or non-hoistable only")
}

// --- helpers used only by Monomorphize above ---

// formatToBytesOrDie formats the file. If formatting fails, it prints the
// offending line/column (and surrounding context) to stdout, then panics
// with the original error.
func formatToBytesOrDie(fset *token.FileSet, file *ast.File) []byte {
	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		debugFormatFailure(fset, file, err)
		panic(err)
	}
	return out.Bytes()
}

func debugFormatFailure(fset *token.FileSet, file *ast.File, err error) {
	fmt.Printf("\n--- format.Node failed: %v ---\n", err)

	// Best-effort print current source using printer (less strict than format.Node).
	var buf bytes.Buffer
	if e := printer.Fprint(&buf, fset, file); e != nil {
		fmt.Println("printer.Fprint also failed; dumping AST instead:")
		var astBuf bytes.Buffer
		_ = ast.Fprint(&astBuf, fset, file, nil)
		fmt.Print(astBuf.String())
		return
	}
	src := buf.Bytes()

	// Pull "line:col" out of the error message (looks like "(334:35:" ...).
	line, col := parseLineCol(err.Error())
	lines := bytes.Split(src, []byte("\n"))

	if line >= 1 && line <= len(lines) {
		from := line - 3
		if from < 1 {
			from = 1
		}
		to := line + 3
		if to > len(lines) {
			to = len(lines)
		}
		for i := from; i <= to; i++ {
			prefix := "   "
			if i == line {
				prefix = ">>>"
			}
			fmt.Printf("%s %6d | %s\n", prefix, i, lines[i-1])
			if i == line && col >= 1 {
				if col > len(lines[i-1]) {
					col = len(lines[i-1])
				}
				spaces := bytes.Repeat([]byte(" "), col-1)
				fmt.Printf("            %s^\n", spaces)
			}
		}
	} else {
		// If we couldn't parse the position, just dump the whole source.
		fmt.Println("--- could not parse line:col from error; dumping source ---")
		fmt.Print(string(src))
		fmt.Println()
	}
}

// parseLineCol extracts the first "(\d+:\d+:" pattern's numbers from msg.
// Returns 0,0 if not found.
func parseLineCol(msg string) (int, int) {
	// Find the first '(' then read digits:digits:
	i := -1
	for idx := 0; idx < len(msg); idx++ {
		if msg[idx] == '(' {
			i = idx + 1
			break
		}
	}
	if i < 0 {
		return 0, 0
	}
	// read line
	line := 0
	for i < len(msg) && msg[i] >= '0' && msg[i] <= '9' {
		line = line*10 + int(msg[i]-'0')
		i++
	}
	if i >= len(msg) || msg[i] != ':' {
		return 0, 0
	}
	i++
	// read col
	col := 0
	for i < len(msg) && msg[i] >= '0' && msg[i] <= '9' {
		col = col*10 + int(msg[i]-'0')
		i++
	}
	// require a trailing ':' like "(334:35:"
	if i >= len(msg) || msg[i] != ':' {
		return 0, 0
	}
	return line, col
}

// ---- Local type hoisting ----

// buildHoistedName creates a stable exported name for a local type.
func buildHoistedName(funcName string, blockPath []int, localName string, taken map[string]bool) string {
	var b strings.Builder
	b.WriteString("Local_")
	if funcName == "" {
		funcName = "anon"
	}
	b.WriteString(funcName)
	for _, idx := range blockPath {
		fmt.Fprintf(&b, "_B%d", idx)
	}
	b.WriteString("_")
	if localName == "" {
		localName = "T"
	}
	// Ensure exported
	if c := localName[0]; c >= 'a' && c <= 'z' {
		localName = strings.ToUpper(localName[:1]) + localName[1:]
	}
	b.WriteString(localName)

	base := b.String()
	name := base
	suf := 2
	for taken[name] {
		name = fmt.Sprintf("%s_%d", base, suf)
		suf++
	}
	taken[name] = true
	return name
}

func collectTopLevelNames(file *ast.File) map[string]bool {
	taken := map[string]bool{}
	for _, d := range file.Decls {
		switch dd := d.(type) {
		case *ast.GenDecl:
			for _, sp := range dd.Specs {
				switch s := sp.(type) {
				case *ast.TypeSpec:
					if s.Name != nil {
						taken[s.Name.Name] = true
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						taken[n.Name] = true
					}
				}
			}
		case *ast.FuncDecl:
			if dd.Name != nil {
				taken[dd.Name.Name] = true
			}
		}
	}
	return taken
}

// hoistLocalTypes finds block-local type declarations in function bodies,
// lifts them to package scope with exported, globally-unique names,
// rewrites all references (Defs/Uses) to the new names, and removes the
// original local declarations. Returns true if the file was modified.
func hoistLocalTypes(file *ast.File, fset *token.FileSet, info *types.Info) bool {
	changed := false
	taken := collectTopLevelNames(file)

	type localDef struct {
		obj      *types.TypeName
		newName  string
		spec     *ast.TypeSpec
		genDecl  *ast.GenDecl
		declStmt *ast.DeclStmt
		block    *ast.BlockStmt
	}

	var locals []*localDef
	localByObj := map[*types.TypeName]*localDef{}

	// Per-function traversal with a block path to build stable exported names.
	var processBlock func(funcName string, blk *ast.BlockStmt, path []int, counter *int)
	processBlock = func(funcName string, blk *ast.BlockStmt, path []int, counter *int) {
		if blk == nil {
			return
		}
		for _, st := range blk.List {
			*counter++
			cur := append(path, *counter)

			// Collect local type specs declared via "type" inside blocks.
			if ds, ok := st.(*ast.DeclStmt); ok {
				if gd, ok := ds.Decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
					for _, sp := range gd.Specs {
						ts, ok := sp.(*ast.TypeSpec)
						if !ok || ts.Name == nil {
							continue
						}
						objI := info.Defs[ts.Name]
						tn, _ := objI.(*types.TypeName)
						if tn == nil {
							continue
						}
						// Skip package-scope types.
						if pkg := tn.Pkg(); pkg != nil && tn.Parent() == pkg.Scope() {
							continue
						}
						// Don't hoist locals that reference type parameters (they'd become undefined).
						if typeExprMentionsTypeParams(ts.Type, info) {
							continue
						}

						newName := buildHoistedName(funcName, cur, ts.Name.Name, taken)
						ld := &localDef{
							obj:      tn,
							newName:  newName,
							spec:     ts,
							genDecl:  gd,
							declStmt: ds,
							block:    blk,
						}
						locals = append(locals, ld)
						localByObj[tn] = ld
					}
				}
			}

			// Recurse into common nested blocks.
			switch s := st.(type) {
			case *ast.BlockStmt:
				processBlock(funcName, s, cur, counter)
			case *ast.IfStmt:
				processBlock(funcName, s.Body, cur, counter)
				if s.Else != nil {
					if eb, ok := s.Else.(*ast.BlockStmt); ok {
						processBlock(funcName, eb, cur, counter)
					} else if es, ok := s.Else.(*ast.IfStmt); ok {
						processBlock(funcName, es.Body, cur, counter)
						if es.Else != nil {
							if eb2, ok := es.Else.(*ast.BlockStmt); ok {
								processBlock(funcName, eb2, cur, counter)
							}
						}
					}
				}
			case *ast.ForStmt:
				processBlock(funcName, s.Body, cur, counter)
			case *ast.RangeStmt:
				processBlock(funcName, s.Body, cur, counter)
			case *ast.SwitchStmt:
				processBlock(funcName, s.Body, cur, counter)
			case *ast.TypeSwitchStmt:
				processBlock(funcName, s.Body, cur, counter)
			case *ast.SelectStmt:
				processBlock(funcName, s.Body, cur, counter)
			}
		}
	}

	// Walk all functions to find locals.
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			fn := ""
			if fd.Name != nil {
				fn = fd.Name.Name
			}
			c := 0
			processBlock(fn, fd.Body, nil, &c)
		}
	}

	if len(locals) == 0 {
		return false
	}

	// 1) Create top-level copies with new names; 2) remember which specs to remove.
	toRemove := map[*ast.TypeSpec]bool{}
	for _, ld := range locals {
		// Deep-clone the underlying type expression to avoid sharing AST nodes
		// between the original local spec and the new top-level spec.
		clonedType := cloneExpr(ld.spec.Type)

		newTS := &ast.TypeSpec{
			Name: ast.NewIdent(ld.newName),
			Type: clonedType,
		}
		newGD := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{newTS}}
		file.Decls = append(file.Decls, newGD)
		toRemove[ld.spec] = true
		changed = true
	}

	// 2) Rewrite references: any Ident that resolves to the local obj -> newName.
	ast.Inspect(file, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || id == nil {
			return true
		}
		if obj, ok := info.Defs[id].(*types.TypeName); ok && obj != nil {
			if ld, ok2 := localByObj[obj]; ok2 {
				id.Name = ld.newName
			}
		}
		if obj, ok := info.Uses[id].(*types.TypeName); ok && obj != nil {
			if ld, ok2 := localByObj[obj]; ok2 {
				id.Name = ld.newName
			}
		}
		return true
	})

	// 3) Remove the original local type specs (and prune empty DeclStmts) per block.
	doneBlk := map[*ast.BlockStmt]bool{}
	for _, ld := range locals {
		if doneBlk[ld.block] {
			continue
		}
		doneBlk[ld.block] = true
		var newList []ast.Stmt
		for _, st := range ld.block.List {
			ds, ok := st.(*ast.DeclStmt)
			if !ok {
				newList = append(newList, st)
				continue
			}
			gd, ok := ds.Decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				newList = append(newList, st)
				continue
			}
			// Filter out hoisted specs in this GenDecl.
			var keep []ast.Spec
			for _, sp := range gd.Specs {
				if ts, ok := sp.(*ast.TypeSpec); ok && toRemove[ts] {
					continue
				}
				keep = append(keep, sp)
			}
			if len(keep) == 0 {
				// Drop the whole DeclStmt.
				continue
			}
			if len(keep) == len(gd.Specs) {
				// Unchanged.
				newList = append(newList, st)
				continue
			}
			ng := *gd
			ng.Specs = keep
			newList = append(newList, &ast.DeclStmt{Decl: &ng})
		}
		ld.block.List = newList
	}

	return changed
}

// ---- helpers ----

func appendDecl(file *ast.File, d ast.Decl) { file.Decls = append(file.Decls, d) }

// Improved: supports X, *X, X[T], *X[T], X[T,U], *X[T,U], with extra parens tolerated.
func baseIdentOfReceiver(recvType ast.Expr) (string, bool) {
	t := recvType
	for {
		if se, ok := t.(*ast.StarExpr); ok {
			t = se.X
			continue
		}
		break
	}
	switch rr := t.(type) {
	case *ast.Ident:
		return rr.Name, true
	case *ast.IndexExpr:
		if id, ok := rr.X.(*ast.Ident); ok {
			return id.Name, true
		}
	case *ast.IndexListExpr:
		if id, ok := rr.X.(*ast.Ident); ok {
			return id.Name, true
		}
	case *ast.ParenExpr:
		return baseIdentOfReceiver(rr.X)
	}
	return "", false
}

func flattenFieldListIdents(fl *ast.FieldList) []string {
	var out []string
	if fl == nil {
		return out
	}
	for _, f := range fl.List {
		for _, n := range f.Names {
			out = append(out, n.Name)
		}
	}
	return out
}

func cleanIdent(s string) string {
	s = strings.ReplaceAll(s, "*", "Ptr")
	s = strings.ReplaceAll(s, "[]", "Slice")
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "]", "")
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, ",", "C")
	s = strings.ReplaceAll(s, "*", "Ptr")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

func cloneExprList(es []ast.Expr) []ast.Expr {
	out := make([]ast.Expr, len(es))
	for i, e := range es {
		out[i] = cloneExpr(e)
	}
	return out
}

// Clone expr via print-parse roundtrip to keep things simple and robust.
func cloneExpr(e ast.Expr) ast.Expr {
	src := nodeToSnippet("package p; var _ = (", e, ")")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "clone.go", src, 0)
	if err != nil {
		panic(err)
	}
	vs := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	par := vs.Values[0].(*ast.ParenExpr)
	return par.X
}

func nodeToSnippet(prefix string, n ast.Node, suffix string) string {
	var buf bytes.Buffer
	buf.WriteString(prefix)
	if err := printer.Fprint(&buf, token.NewFileSet(), n); err != nil {
		panic(err)
	}
	buf.WriteString(suffix)
	return buf.String()
}

func cloneTypeDecl(ts *ast.TypeSpec, newName string, subst map[string]ast.Expr) ast.Decl {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), ts)
	src := "package p; type " + buf.String()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "ty.go", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	gd := f.Decls[0].(*ast.GenDecl)
	cp := gd.Specs[0].(*ast.TypeSpec)

	applySubstToNode(cp, subst)
	cp.TypeParams = nil
	cp.Name = ast.NewIdent(newName)

	out := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{cp}}
	return out
}

func cloneMethodForConcrete(fd *ast.FuncDecl, baseName, concreteName string, subst map[string]ast.Expr) *ast.FuncDecl {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), fd)
	src := "package p; " + buf.String()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "m.go", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	clone := f.Decls[0].(*ast.FuncDecl)

	if clone.Recv == nil || len(clone.Recv.List) != 1 {
		panic("unexpected receiver shape")
	}
	recv := clone.Recv.List[0]
	isPtr := false
	t := recv.Type
	if se, ok := t.(*ast.StarExpr); ok {
		isPtr = true
		t = se.X
	}
	var newBase ast.Expr = ast.NewIdent(concreteName)
	if isPtr {
		newBase = &ast.StarExpr{X: newBase}
	}
	recv.Type = newBase

	applySubstToNode(clone, subst)

	return clone
}

func cloneFuncForConcrete(fd *ast.FuncDecl, newName string, subst map[string]ast.Expr) *ast.FuncDecl {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), fd)
	src := "package p; " + buf.String()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "f.go", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	clone := f.Decls[0].(*ast.FuncDecl)

	if clone.Type != nil {
		clone.Type.TypeParams = nil
	}
	clone.Name = ast.NewIdent(newName)

	applySubstToNode(clone, subst)

	return clone
}

// applySubstToNode walks nn and applies 'subst' where appropriate.
// Important: we do NOT rewrite method callees like (update).ApplyUpdate(...).
func applySubstToNode(n ast.Node, subst map[string]ast.Expr) {
	if n == nil {
		return
	}

	ast.Inspect(n, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.Field:
			x.Type = substInExpr(x.Type, subst)

		case *ast.ValueSpec:
			if x.Type != nil {
				x.Type = substInExpr(x.Type, subst)
			}
			for i, v := range x.Values {
				x.Values[i] = substInExpr(v, subst)
			}

		case *ast.TypeSpec:
			x.Type = substInExpr(x.Type, subst)

		case *ast.ReturnStmt:
			for i, r := range x.Results {
				x.Results[i] = substInExpr(r, subst)
			}

		case *ast.AssignStmt:
			for i, r := range x.Rhs {
				x.Rhs[i] = substInExpr(r, subst)
			}

		case *ast.CompositeLit:
			// Also substitute inside the literal's type.
			x.Type = substInExpr(x.Type, subst)
			for i, e := range x.Elts {
				x.Elts[i] = substInExpr(e, subst)
			}

		case *ast.UnaryExpr:
			// Handle &T{...} and similar.
			x.X = substInExpr(x.X, subst)

		case *ast.CallExpr:
			// Do NOT rewrite method callees: foo.Bar(...) must remain untouched.
			// Only rewrite the callee if it syntactically looks like a type expr
			// (i.e., a conversion), e.g. (value)(x), (*value)(x), value(x).
			if _, isSel := x.Fun.(*ast.SelectorExpr); !isSel && looksLikeTypeExpr(x.Fun) {
				x.Fun = substInExpr(x.Fun, subst)
			}
			// Always rewrite arguments.
			for i, a := range x.Args {
				x.Args[i] = substInExpr(a, subst)
			}

		case *ast.IndexExpr:
			x.X = substInExpr(x.X, subst)
			x.Index = substInExpr(x.Index, subst)

		case *ast.IndexListExpr:
			x.X = substInExpr(x.X, subst)
			for i, a := range x.Indices {
				x.Indices[i] = substInExpr(a, subst)
			}

		case *ast.StarExpr:
			x.X = substInExpr(x.X, subst)

		case *ast.ParenExpr:
			x.X = substInExpr(x.X, subst)
		}
		return true
	})
}

// looksLikeTypeExpr reports whether e is syntactically a type expression
// composed only of type-ish nodes (Ident, Selector over pkg Ident, Index, Star,
// Paren, etc.). This lets us safely rewrite conversions like value(x) or
// (*value)(x) without touching value-method calls like (update).ApplyUpdate.
func looksLikeTypeExpr(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.Ident:
		return true
	case *ast.ParenExpr:
		return looksLikeTypeExpr(t.X)
	case *ast.StarExpr:
		return looksLikeTypeExpr(t.X)
	case *ast.ArrayType:
		// Conversion callee is never an array type; keep false to be conservative.
		return false
	case *ast.ChanType, *ast.MapType, *ast.StructType, *ast.InterfaceType, *ast.FuncType:
		// These can be types, but they don't occur as conversion callees in normal code.
		// Keep false to avoid over-matching; extend if you really need them.
		return false
	case *ast.SelectorExpr:
		// Qualifed type like pkg.Type or pkg.Generic[T]
		// Only treat as type-ish if X itself is an Ident (a package name) or
		// recursively type-ish (for nested pkg paths).
		return looksLikeTypeExpr(t.X)
	case *ast.IndexExpr:
		return looksLikeTypeExpr(t.X) && looksLikeTypeExpr(t.Index)
	case *ast.IndexListExpr:
		if !looksLikeTypeExpr(t.X) {
			return false
		}
		for _, a := range t.Indices {
			if !looksLikeTypeExpr(a) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// substInExpr applies 'subst' to a type/value expression where safe.
// Important: for SelectorExpr, if the left side is a type parameter name,
// we do NOT rewrite it; otherwise (update).ApplyUpdate would become
// (lazy).ApplyUpdate (incorrect).
func substInExpr(e ast.Expr, subst map[string]ast.Expr) ast.Expr {
	switch t := e.(type) {
	case *ast.Ident:
		if repl, ok := subst[t.Name]; ok {
			return cloneExpr(repl)
		}
		return e

	case *ast.ParenExpr:
		t.X = substInExpr(t.X, subst)
		return t

	case *ast.StarExpr:
		t.X = substInExpr(t.X, subst)
		return t

	case *ast.ArrayType:
		if t.Len != nil {
			t.Len = substInExpr(t.Len, subst)
		}
		t.Elt = substInExpr(t.Elt, subst)
		return t

	case *ast.MapType:
		t.Key = substInExpr(t.Key, subst)
		t.Value = substInExpr(t.Value, subst)
		return t

	case *ast.ChanType:
		t.Value = substInExpr(t.Value, subst)
		return t

	case *ast.SelectorExpr:
		// If the left side is exactly a type parameter identifier that appears
		// in 'subst', treat it as a value receiver/expr and DO NOT rewrite it.
		if id, ok := t.X.(*ast.Ident); ok {
			if _, isTP := subst[id.Name]; isTP {
				return t
			}
		}
		t.X = substInExpr(t.X, subst)
		return t

	case *ast.IndexExpr:
		t.X = substInExpr(t.X, subst)
		t.Index = substInExpr(t.Index, subst)
		return t

	case *ast.IndexListExpr:
		t.X = substInExpr(t.X, subst)
		for i, a := range t.Indices {
			t.Indices[i] = substInExpr(a, subst)
		}
		return t

	case *ast.StructType:
		if t.Fields != nil {
			for _, f := range t.Fields.List {
				f.Type = substInExpr(f.Type, subst)
			}
		}
		return t

	case *ast.InterfaceType:
		if t.Methods != nil {
			for _, f := range t.Methods.List {
				if ft, ok := f.Type.(*ast.FuncType); ok {
					f.Type = substInFuncType(ft, subst)
				} else {
					f.Type = substInExpr(f.Type, subst)
				}
			}
		}
		return t

	case *ast.FuncType:
		return substInFuncType(t, subst)

	default:
		return e
	}
}

func substInFuncType(ft *ast.FuncType, subst map[string]ast.Expr) *ast.FuncType {
	var params, results *ast.FieldList
	if ft.Params != nil {
		params = &ast.FieldList{}
		for _, f := range ft.Params.List {
			ff := *f
			ff.Type = substInExpr(f.Type, subst)
			params.List = append(params.List, &ff)
		}
	}
	if ft.Results != nil {
		results = &ast.FieldList{}
		for _, f := range ft.Results.List {
			ff := *f
			ff.Type = substInExpr(f.Type, subst)
			results.List = append(results.List, &ff)
		}
	}
	return &ast.FuncType{
		Params:     params,
		Results:    results,
		TypeParams: nil,
	}
}

// rewriteInstantiations replaces explicit Foo[Args...] and generic calls with concrete identifiers.
func rewriteInstantiations(file *ast.File, created map[instKey]string, callKeys map[*ast.CallExpr]instKey) bool {
	changed := false

	var parents []ast.Node
	replaceInParent := func(parent ast.Node, old ast.Expr, newName string) bool {
		newIdent := ast.NewIdent(newName)
		return setChildExprInParent(parent, old, newIdent)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			if len(parents) > 0 {
				parents = parents[:len(parents)-1]
			}
			return false
		}
		if len(parents) > 0 {
			parent := parents[len(parents)-1]

			switch ix := n.(type) {
			case *ast.IndexListExpr:
				var baseName string
				switch b := ix.X.(type) {
				case *ast.Ident:
					baseName = b.Name
				case *ast.SelectorExpr:
					baseName = b.Sel.Name
				}
				if baseName != "" {
					k := instKey{Name: baseName, Args: argsKey(ix.Indices)}
					if newName, ok := created[k]; ok {
						if replaceInParent(parent, ix, newName) {
							changed = true
							return false
						}
					}
				}

			case *ast.IndexExpr:
				var baseName string
				switch b := ix.X.(type) {
				case *ast.Ident:
					baseName = b.Name
				case *ast.SelectorExpr:
					baseName = b.Sel.Name
				}
				if baseName != "" {
					k := instKey{Name: baseName, Args: argsKey([]ast.Expr{ix.Index})}
					if newName, ok := created[k]; ok {
						if replaceInParent(parent, ix, newName) {
							changed = true
							return false
						}
					}
				}
			}
		}

		if call, ok := n.(*ast.CallExpr); ok {
			if id, ok2 := call.Fun.(*ast.Ident); ok2 {
				if k, ok3 := callKeys[call]; ok3 {
					if newName, ok4 := created[k]; ok4 {
						id.Name = newName
						changed = true
					}
				}
			}
		}

		parents = append(parents, n)
		return true
	})

	return changed
}

func setChildExprInParent(parent ast.Node, old, newE ast.Expr) bool {
	switch p := parent.(type) {
	case *ast.ExprStmt:
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.AssignStmt:
		for i := range p.Lhs {
			if p.Lhs[i] == old {
				p.Lhs[i] = newE
				return true
			}
		}
		for i := range p.Rhs {
			if p.Rhs[i] == old {
				p.Rhs[i] = newE
				return true
			}
		}
	case *ast.ReturnStmt:
		for i := range p.Results {
			if p.Results[i] == old {
				p.Results[i] = newE
				return true
			}
		}
	case *ast.SendStmt:
		if p.Chan == old {
			p.Chan = newE
			return true
		}
		if p.Value == old {
			p.Value = newE
			return true
		}
	case *ast.RangeStmt:
		if p.Key == old {
			p.Key = newE
			return true
		}
		if p.Value == old {
			p.Value = newE
			return true
		}
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.IncDecStmt:
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.SwitchStmt:
		if p.Tag == old {
			p.Tag = newE
			return true
		}
	case *ast.CaseClause:
		for i := range p.List {
			if p.List[i] == old {
				p.List[i] = newE
				return true
			}
		}

	case *ast.CallExpr:
		if p.Fun == old {
			p.Fun = newE
			return true
		}
		for i := range p.Args {
			if p.Args[i] == old {
				p.Args[i] = newE
				return true
			}
		}
	case *ast.SelectorExpr:
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.ParenExpr:
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.StarExpr:
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.UnaryExpr:
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.BinaryExpr:
		if p.X == old {
			p.X = newE
			return true
		}
		if p.Y == old {
			p.Y = newE
			return true
		}
	case *ast.SliceExpr:
		if p.X == old {
			p.X = newE
			return true
		}
		if p.Low == old {
			p.Low = newE
			return true
		}
		if p.High == old {
			p.High = newE
			return true
		}
		if p.Max == old {
			p.Max = newE
			return true
		}
	case *ast.IndexExpr:
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.IndexListExpr:
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.TypeAssertExpr:
		if p.X == old {
			p.X = newE
			return true
		}
	case *ast.KeyValueExpr:
		if p.Key == old {
			p.Key = newE
			return true
		}
		if p.Value == old {
			p.Value = newE
			return true
		}
	case *ast.CompositeLit:
		if p.Type == old {
			p.Type = newE
			return true
		}
		for i := range p.Elts {
			if p.Elts[i] == old {
				p.Elts[i] = newE
				return true
			}
		}
	case *ast.TypeSpec:
		if p.Type == old {
			p.Type = newE
			return true
		}
	case *ast.ValueSpec:
		if p.Type == old {
			p.Type = newE
			return true
		}
		for i := range p.Values {
			if p.Values[i] == old {
				p.Values[i] = newE
				return true
			}
		}
	case *ast.Field:
		if p.Type == old {
			p.Type = newE
			return true
		}
	case *ast.ArrayType:
		if p.Elt == old {
			p.Elt = newE
			return true
		}
	case *ast.MapType:
		if p.Key == old {
			p.Key = newE
			return true
		}
		if p.Value == old {
			p.Value = newE
			return true
		}
	case *ast.ChanType:
		if p.Value == old {
			p.Value = newE
			return true
		}
	}
	return false
}

// Convert a types.Type into an ast.Expr by printing the type and parsing it back.
func typeToExprWithFile(file *ast.File, t types.Type) ast.Expr {
	pkgName := file.Name.Name
	src := types.TypeString(t, func(p *types.Package) string {
		if p == nil || p.Name() == pkgName {
			return ""
		}
		return p.Name()
	})
	e, err := parser.ParseExpr(src)
	if err != nil {
		panic(fmt.Errorf("cannot parse type expr %q: %w", src, err))
	}
	return e
}

func getFuncDecl(file *ast.File, genFuncs map[string]*ast.FuncDecl, name string) *ast.FuncDecl {
	if fd, ok := genFuncs[name]; ok && fd != nil {
		return fd
	}
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Name != nil && f.Name.Name == name {
			return f
		}
	}
	return nil
}

// --------- hoistability (types-based) ---------

func isHoistableType(t types.Type) bool {
	seen := map[types.Type]bool{}
	var rec func(types.Type) bool
	rec = func(tt types.Type) bool {
		if tt == nil {
			return true
		}
		if seen[tt] {
			return true
		}
		seen[tt] = true

		switch x := tt.(type) {
		case *types.Basic:
			return true
		case *types.Pointer:
			return rec(x.Elem())
		case *types.Slice:
			return rec(x.Elem())
		case *types.Array:
			return rec(x.Elem())
		case *types.Map:
			return rec(x.Key()) && rec(x.Elem())
		case *types.Chan:
			return rec(x.Elem())
		case *types.Tuple:
			for i := 0; i < x.Len(); i++ {
				if !rec(x.At(i).Type()) {
					return false
				}
			}
			return true
		case *types.Signature:
			p := x.Params()
			for i := 0; i < p.Len(); i++ {
				if !rec(p.At(i).Type()) {
					return false
				}
			}
			r := x.Results()
			for i := 0; i < r.Len(); i++ {
				if !rec(r.At(i).Type()) {
					return false
				}
			}
			return true
		case *types.Interface:
			for i := 0; i < x.NumEmbeddeds(); i++ {
				if !rec(x.EmbeddedType(i)) {
					return false
				}
			}
			for i := 0; i < x.NumMethods(); i++ {
				if !rec(x.Method(i).Type()) {
					return false
				}
			}
			return true
		case *types.Struct:
			for i := 0; i < x.NumFields(); i++ {
				if !rec(x.Field(i).Type()) {
					return false
				}
			}
			return true
		case *types.Named:
			obj := x.Obj()
			if obj == nil || obj.Pkg() == nil {
				return true
			}
			if obj.Parent() != obj.Pkg().Scope() {
				return false
			}
			return rec(x.Underlying())
		case *types.TypeParam:
			return false
		default:
			return rec(x.Underlying())
		}
	}
	return rec(t)
}

func predeclTypeByName(name string) types.Type {
	switch name {
	case "bool":
		return types.Typ[types.Bool]
	case "byte":
		return types.Typ[types.Byte]
	case "rune":
		return types.Typ[types.Rune]
	case "string":
		return types.Typ[types.String]
	case "uintptr":
		return types.Typ[types.Uintptr]
	case "int":
		return types.Typ[types.Int]
	case "int8":
		return types.Typ[types.Int8]
	case "int16":
		return types.Typ[types.Int16]
	case "int32":
		return types.Typ[types.Int32]
	case "int64":
		return types.Typ[types.Int64]
	case "uint":
		return types.Typ[types.Uint]
	case "uint8":
		return types.Typ[types.Uint8]
	case "uint16":
		return types.Typ[types.Uint16]
	case "uint32":
		return types.Typ[types.Uint32]
	case "uint64":
		return types.Typ[types.Uint64]
	case "float32":
		return types.Typ[types.Float32]
	case "float64":
		return types.Typ[types.Float64]
	case "complex64":
		return types.Typ[types.Complex64]
	case "complex128":
		return types.Typ[types.Complex128]
	case "any":
		if tn, ok := types.Universe.Lookup("any").(*types.TypeName); ok {
			return tn.Type()
		}
		return nil
	case "error":
		if tn, ok := types.Universe.Lookup("error").(*types.TypeName); ok {
			return tn.Type()
		}
		return nil
	default:
		return nil
	}
}

// typeExprMentionsTypeParams returns true if the type expression references any type parameters.
func typeExprMentionsTypeParams(expr ast.Expr, info *types.Info) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found || n == nil {
			return false
		}
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		if obj, ok := info.Uses[id].(*types.TypeName); ok && obj != nil {
			if _, isTP := obj.Type().(*types.TypeParam); isTP {
				found = true
				return false
			}
		}
		if obj, ok := info.Defs[id].(*types.TypeName); ok && obj != nil {
			if _, isTP := obj.Type().(*types.TypeParam); isTP {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// Ensures a valid Go identifier: letters/digits/underscore only; if it
// doesn't start with a letter or '_', prefixes 'X'.
func safeIdent(s string) string {
	// First apply the existing cleanup rules you already rely on.
	s = cleanIdent(s)

	// Enforce identifier charset.
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "X"
	}
	// Must start with letter or underscore.
	h := out[0]
	if !((h >= 'a' && h <= 'z') || (h >= 'A' && h <= 'Z') || h == '_') {
		out = append([]rune{'X'}, out...)
	}
	return string(out)
}

// Build a stable key for a list of type args WITHOUT commas (use 'C' as a separator).
func argsKey(args []ast.Expr) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		parts = append(parts, typeExprToStableName(a))
	}
	// 'C' is already used in your cleanIdent() mapping; it’s safe and comma-free.
	return strings.Join(parts, "C")
}

// Convert a type expression to a stable, comma-free, identifier-safe name.
func typeExprToStableName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return safeIdent(t.Name)
	case *ast.SelectorExpr:
		// pkg.Type -> <pkg>_<Type>
		return safeIdent(typeExprToStableName(t.X) + "_" + t.Sel.Name)
	case *ast.StarExpr:
		return safeIdent("Ptr" + typeExprToStableName(t.X))
	case *ast.ArrayType:
		if t.Len == nil {
			return safeIdent("Slice" + typeExprToStableName(t.Elt))
		}
		return safeIdent("Arr" + typeExprToStableName(t.Elt))
	case *ast.MapType:
		return safeIdent("Map" + typeExprToStableName(t.Key) + "To" + typeExprToStableName(t.Value))
	case *ast.ChanType:
		return safeIdent("Chan" + typeExprToStableName(t.Value))
	case *ast.IndexListExpr:
		// E.g. Foo[T, U] -> FooOf<T_name>C<U_name> (no commas)
		return safeIdent(typeExprToStableName(t.X) + "Of" + argsKey(t.Indices))
	case *ast.IndexExpr:
		// E.g. Foo[T] -> FooOf<T_name> (no commas)
		return safeIdent(typeExprToStableName(t.X) + "Of" + typeExprToStableName(t.Index))
	case *ast.FuncType:
		return "Func"
	case *ast.StructType:
		return "Struct"
	case *ast.InterfaceType:
		return "Iface"
	default:
		// Fallback: print and sanitize.
		var buf bytes.Buffer
		_ = printer.Fprint(&buf, token.NewFileSet(), e)
		return safeIdent(buf.String())
	}
}

// Build the concrete symbol name: Base + G1<...>G2<...>..., then sanitize once more.
func makeMonoName(base string, args []ast.Expr) string {
	var parts []string
	for i, a := range args {
		parts = append(parts, fmt.Sprintf("G%d%s", i+1, typeExprToStableName(a)))
	}
	return safeIdent(base + strings.Join(parts, ""))
}

// debugTypecheckFailure prints a highlighted snippet for the first position
// contained in a go/types error, falling back to parsing the error string.
func debugTypecheckFailure(fset *token.FileSet, file *ast.File, err error) {
	fmt.Printf("\n--- types.Check failed: %v ---\n", err)

	// Produce best-effort source from the AST.
	var buf bytes.Buffer
	if e := printer.Fprint(&buf, fset, file); e != nil {
		fmt.Println("printer.Fprint also failed; dumping AST instead:")
		var astBuf bytes.Buffer
		_ = ast.Fprint(&astBuf, fset, file, nil)
		fmt.Print(astBuf.String())
		return
	}
	src := buf.Bytes()
	lines := bytes.Split(src, []byte("\n"))

	// Try to get line/col from concrete error types.
	line, col := 0, 0

	// Case 1: a single types.Error
	if te, ok := err.(types.Error); ok {
		pos := fset.Position(te.Pos)
		line, col = pos.Line, pos.Column
	}

	// Case 2: sometimes the error message is like "file.go:line:col: msg"
	if line == 0 || col == 0 {
		if l2, c2 := parseLineCol(err.Error()); l2 > 0 && c2 > 0 {
			line, col = l2, c2
		}
	}

	// If we still don't have a position, just dump the whole source.
	if line <= 0 || col <= 0 || line > len(lines) {
		fmt.Println("--- could not determine position from error; dumping source ---")
		fmt.Print(string(src))
		fmt.Println()
		return
	}

	from := line - 10
	if from < 1 {
		from = 1
	}
	to := line + 10
	if to > len(lines) {
		to = len(lines)
	}
	for i := from; i <= to; i++ {
		prefix := "   "
		if i == line {
			prefix = ">>>"
		}
		fmt.Printf("%s %6d | %s\n", prefix, i, lines[i-1])
		if i == line {
			if col > len(lines[i-1]) {
				col = len(lines[i-1])
			}
			if col < 1 {
				col = 1
			}
			spaces := bytes.Repeat([]byte(" "), col-1)
			fmt.Printf("            %s^\n", spaces)
		}
	}
}
