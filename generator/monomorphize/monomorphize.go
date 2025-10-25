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

// MonomorphizeIterative does what Monomorphize does, but performs it in small
// steps: on each iteration it materializes at most one instantiation and
// rewrites its callsites/usages, then regenerates the source and reparses it.
// This favors correctness over speed on small binaries.
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

		// Index generic decls / methods (same as your eager version)
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

		// Type-check current file
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
			Error:    func(err error) { panic(err) },
		}
		if _, err := conf.Check(file.Name.Name, fset, []*ast.File{file}, info); err != nil {
			panic(err)
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

			// Rewrite usages for THIS instantiation only
			toRewrite := map[instKey]string{k: concreteName}
			rewritten := rewriteInstantiations(file, toRewrite, callKeys)
			_ = rewritten // we expect it to be true in practice

			// Emit fresh source and restart the outer loop with a new parse/typecheck.
			var out bytes.Buffer
			if err := format.Node(&out, fset, file); err != nil {
				panic(err)
			}
			src = out.Bytes()
			madeChange = true
			break // <-- enforce "one replacement per iteration"
		}

		if !madeChange {
			// Nothing left to do in this iteration => done.
			return src
		}
	}

	panic("MonomorphizeIterative: exceeded iteration limit; possible cycle or non-hoistable only")
}

// ---- helpers ----

func appendDecl(file *ast.File, d ast.Decl) { file.Decls = append(file.Decls, d) }

// Improved: supports X, *X, X[T], *X[T], X[T,U], *X[T,U], with extra parens tolerated.
func baseIdentOfReceiver(recvType ast.Expr) (string, bool) {
	// Strip pointer layers
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

func argsKey(args []ast.Expr) string {
	var b strings.Builder
	for i, a := range args {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(typeExprToStableName(a))
	}
	return b.String()
}

func makeMonoName(base string, args []ast.Expr) string {
	var parts []string
	for i, a := range args {
		parts = append(parts, fmt.Sprintf("G%d%s", i+1, typeExprToStableName(a)))
	}
	return base + strings.Join(parts, "")
}

func typeExprToStableName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return cleanIdent(t.Name)
	case *ast.SelectorExpr:
		return typeExprToStableName(t.X) + "_" + cleanIdent(t.Sel.Name)
	case *ast.StarExpr:
		return "Ptr" + typeExprToStableName(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "Slice" + typeExprToStableName(t.Elt)
		}
		return "Arr" + typeExprToStableName(t.Elt)
	case *ast.MapType:
		return "Map" + typeExprToStableName(t.Key) + "To" + typeExprToStableName(t.Value)
	case *ast.ChanType:
		return "Chan" + typeExprToStableName(t.Value)
	case *ast.IndexListExpr:
		return typeExprToStableName(t.X) + "Of" + argsKey(t.Indices)
	case *ast.IndexExpr:
		return typeExprToStableName(t.X) + "Of" + typeExprToStableName(t.Index)
	case *ast.FuncType:
		return "Func"
	case *ast.StructType:
		return "Struct"
	case *ast.InterfaceType:
		return "Iface"
	default:
		var buf bytes.Buffer
		_ = printer.Fprint(&buf, token.NewFileSet(), e)
		return cleanIdent(buf.String())
	}
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
	// print original spec and reparse to get a deep copy
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

	// Substitute type params throughout
	applySubstToNode(cp, subst)

	// Drop type params and rename
	cp.TypeParams = nil
	cp.Name = ast.NewIdent(newName)

	// Wrap back into a GenDecl
	out := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{cp}}
	return out
}

func cloneMethodForConcrete(fd *ast.FuncDecl, baseName, concreteName string, subst map[string]ast.Expr) *ast.FuncDecl {
	// clone decl
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), fd)
	src := "package p; " + buf.String()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "m.go", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	clone := f.Decls[0].(*ast.FuncDecl)

	// fix receiver type to concrete name (preserve pointer/non-pointer as in original)
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

	// Substitute inside method body/signature
	applySubstToNode(clone, subst)

	return clone
}

func cloneFuncForConcrete(fd *ast.FuncDecl, newName string, subst map[string]ast.Expr) *ast.FuncDecl {
	// clone
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), fd)
	src := "package p; " + buf.String()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "f.go", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	clone := f.Decls[0].(*ast.FuncDecl)

	// drop type params and rename
	if clone.Type != nil {
		clone.Type.TypeParams = nil
	}
	clone.Name = ast.NewIdent(newName)

	// Substitute throughout
	applySubstToNode(clone, subst)

	return clone
}

// applySubstToNode replaces identifiers that match any key in subst with the
// corresponding concrete type expression in common *type positions*.
//
// Besides fields/specs/type specs and composite-literal types, this also
// handles:
//   - builtin `make`/`new`: first argument is a type expression
//   - type conversions: the callee expression itself is a type (e.g. T(x))
//
// In all these spots we run `substInExpr` to swap type params (e.g., K,V,H)
// with their concrete type expressions.
func applySubstToNode(n ast.Node, subst map[string]ast.Expr) {
	ast.Inspect(n, func(nn ast.Node) bool {
		switch x := nn.(type) {
		case *ast.Field:
			if x.Type != nil {
				x.Type = substInExpr(x.Type, subst)
			}
		case *ast.ValueSpec:
			if x.Type != nil {
				x.Type = substInExpr(x.Type, subst)
			}
		case *ast.TypeSpec:
			if x.Type != nil {
				x.Type = substInExpr(x.Type, subst)
			}
		case *ast.CompositeLit:
			// e.g. &HashTable[K,H,V]{ ... }
			if x.Type != nil {
				x.Type = substInExpr(x.Type, subst)
			}
		case *ast.CallExpr:
			// Handle type conversions: Foo[T](v) or T(v)
			// (when Fun denotes a type expression)
			x.Fun = substInExpr(x.Fun, subst)

			// Handle builtins that take a type as the first arg.
			if id, ok := x.Fun.(*ast.Ident); ok {
				if id.Name == "make" || id.Name == "new" {
					if len(x.Args) > 0 {
						x.Args[0] = substInExpr(x.Args[0], subst) // e.g. make([]hashTableEntry[K,V], n)
					}
				}
			}
		}
		return true
	})
}

func substInExpr(e ast.Expr, subst map[string]ast.Expr) ast.Expr {
	switch t := e.(type) {
	case *ast.Ident:
		if repl, ok := subst[t.Name]; ok {
			return cloneExpr(repl)
		}
		return e
	case *ast.StarExpr:
		return &ast.StarExpr{X: substInExpr(t.X, subst)}
	case *ast.ArrayType:
		var ln ast.Expr
		if t.Len != nil {
			ln = t.Len
		}
		return &ast.ArrayType{Len: ln, Elt: substInExpr(t.Elt, subst)}
	case *ast.MapType:
		return &ast.MapType{Key: substInExpr(t.Key, subst), Value: substInExpr(t.Value, subst)}
	case *ast.ChanType:
		return &ast.ChanType{Dir: t.Dir, Value: substInExpr(t.Value, subst)}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{X: substInExpr(t.X, subst), Sel: t.Sel}
	case *ast.IndexExpr:
		return &ast.IndexExpr{X: substInExpr(t.X, subst), Index: substInExpr(t.Index, subst)}
	case *ast.IndexListExpr:
		idxs := make([]ast.Expr, len(t.Indices))
		for i, a := range t.Indices {
			idxs[i] = substInExpr(a, subst)
		}
		return &ast.IndexListExpr{X: substInExpr(t.X, subst), Indices: idxs}
	case *ast.StructType:
		fs := &ast.FieldList{}
		if t.Fields != nil {
			for _, f := range t.Fields.List {
				ff := *f
				ff.Type = substInExpr(f.Type, subst)
				fs.List = append(fs.List, &ff)
			}
		}
		return &ast.StructType{Fields: fs, Incomplete: t.Incomplete}
	case *ast.InterfaceType:
		ms := &ast.FieldList{}
		if t.Methods != nil {
			for _, f := range t.Methods.List {
				ff := *f
				if ft, ok := f.Type.(*ast.FuncType); ok {
					ff.Type = substInFuncType(ft, subst)
				} else {
					ff.Type = substInExpr(f.Type, subst)
				}
				ms.List = append(ms.List, &ff)
			}
		}
		return &ast.InterfaceType{Methods: ms, Incomplete: t.Incomplete}
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

// rewriteInstantiations replaces:
//   - explicit Foo[Args...] nodes (ident or selector base) with the concrete identifier
//   - calls with inferred args: PrintSlice(x) -> PrintSliceG1int(x)
func rewriteInstantiations(file *ast.File, created map[instKey]string, callKeys map[*ast.CallExpr]instKey) bool {
	changed := false

	// Parent stack for ast.Inspect (only needed for replacing Index{,List}Expr)
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
				// Foo[...], or pkg.Foo[...]
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
				// Foo[T], or pkg.Foo[T]
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

		// Also handle callsites with inferred args directly on the node.
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

// setChildExprInParent swaps child expression 'old' with 'newE' inside 'parent'.
// Only expression-typed children are considered (no stmt fields).
func setChildExprInParent(parent ast.Node, old, newE ast.Expr) bool {
	switch p := parent.(type) {
	// ---- statements carrying expressions (expr-only parts) ----
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

	// ---- expressions with expr children ----
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

	// ---- type positions (types are ast.Expr) ----
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
// Types from the same package as 'file' are left unqualified (X[T] not main.X[T]).
func typeToExprWithFile(file *ast.File, t types.Type) ast.Expr {
	pkgName := file.Name.Name
	src := types.TypeString(t, func(p *types.Package) string {
		if p == nil || p.Name() == pkgName {
			return "" // no qualifier for same package
		}
		return p.Name()
	})
	e, err := parser.ParseExpr(src)
	if err != nil {
		panic(fmt.Errorf("cannot parse type expr %q: %w", src, err))
	}
	return e
}

// getFuncDecl returns the *ast.FuncDecl by name, preferring the genFuncs index but
// falling back to a linear scan of the file if necessary.
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

// isHoistableType returns true if t mentions no block-local named types.
// Predeclareds and package-level named types are OK; everything else must
// recursively be OK as well.
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
			// Predeclareds are *Basic, but be defensive:
			if obj == nil || obj.Pkg() == nil {
				return true
			}
			// OK iff declared in package scope of its package.
			if obj.Parent() != obj.Pkg().Scope() {
				return false
			}
			// Also check underlying just in case.
			return rec(x.Underlying())
		case *types.TypeParam:
			// Type params themselves are not hoistable as concrete args.
			return false
		default:
			return rec(x.Underlying())
		}
	}
	return rec(t)
}

// predeclTypeByName returns a predeclared type by name (e.g., "int", "string", "any", "error").
// Returns nil if the name isn't a predeclared/builtin type.
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
		// alias to interface{}
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
