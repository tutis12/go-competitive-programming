package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
)

func PruneUnused(src []byte) []byte {
	// NOTE: Requires these imports in your file:
	//   "bytes", "go/ast", "go/format", "go/importer", "go/parser",
	//   "go/printer", "go/token", "go/types", "strings"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "merged.go", src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return src
	}

	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Implicits:  make(map[ast.Node]types.Object),
	}
	conf := &types.Config{Importer: importer.Default()}
	_, err = conf.Check(file.Name.Name, fset, []*ast.File{file}, info)
	if err != nil {
		// If it doesn't type-check (transient while you iterate), best effort: return as-is.
		return src
	}

	// ---------- Index top-level declarations ----------
	typeDeclByObj := map[*types.TypeName]*ast.TypeSpec{}
	funcDeclByObj := map[*types.Func]*ast.FuncDecl{}
	methodsByName := map[string][]*ast.FuncDecl{} // for fallback printing order
	varDeclSpecs := map[*ast.ValueSpec]token.Token{}
	valueNameToSpec := map[string]*ast.ValueSpec{}
	importDecls := []*ast.GenDecl{}
	var mainFunc *ast.FuncDecl

	for _, d := range file.Decls {
		switch dd := d.(type) {
		case *ast.GenDecl:
			switch dd.Tok {
			case token.IMPORT:
				importDecls = append(importDecls, dd)
			case token.TYPE:
				for _, sp := range dd.Specs {
					ts := sp.(*ast.TypeSpec)
					if obj, ok := info.Defs[ts.Name].(*types.TypeName); ok && obj != nil {
						typeDeclByObj[obj] = ts
					}
				}
			case token.VAR, token.CONST:
				for _, sp := range dd.Specs {
					vs := sp.(*ast.ValueSpec)
					varDeclSpecs[vs] = dd.Tok
					for _, nm := range vs.Names {
						valueNameToSpec[nm.Name] = vs
					}
				}
			}
		case *ast.FuncDecl:
			if dd.Recv == nil {
				if dd.Name.Name == "main" {
					mainFunc = dd
				}
				if obj, ok := info.Defs[dd.Name].(*types.Func); ok && obj != nil {
					funcDeclByObj[obj] = dd
				}
			} else {
				methodsByName[dd.Name.Name] = append(methodsByName[dd.Name.Name], dd)
				if obj, ok := info.Defs[dd.Name].(*types.Func); ok && obj != nil {
					funcDeclByObj[obj] = dd // methods are funcs too
				}
			}
		}
	}

	// ---------- Reachability sets ----------
	reachFunc := map[*types.Func]bool{}     // reachable (free funcs + methods)
	reachType := map[*types.TypeName]bool{} // reachable named types
	reachIface := map[*types.Interface]bool{}
	reachNamed := map[*types.Named]bool{} // concrete named types seen
	reachValue := map[*ast.ValueSpec]bool{}

	// Helper: record a named type (and its *types.Named)
	pushType := func(tn *types.TypeName) {
		if tn == nil || reachType[tn] {
			return
		}
		reachType[tn] = true
		if nm, ok := tn.Type().(*types.Named); ok && nm != nil {
			reachNamed[nm] = true
		}
	}

	// Harvest all types (incl. interfaces) seen in an expression/type position.
	harvestTypes := func(e ast.Expr) {
		if e == nil {
			return
		}
		ast.Inspect(e, func(n ast.Node) bool {
			te, ok := n.(ast.Expr)
			if !ok {
				return true
			}
			tv, ok := info.Types[te]
			if !ok || tv.Type == nil {
				return true
			}
			tt := tv.Type
			// Named?
			if nm, ok := tt.(*types.Named); ok && nm != nil {
				if tn := nm.Obj(); tn != nil {
					pushType(tn)
				}
			}
			// Underlying interface?
			if it, ok := tt.Underlying().(*types.Interface); ok && it != nil {
				reachIface[it] = true
			}
			return true
		})
	}

	// Add all types occurring in a func signature (params/results/recv)
	harvestFuncSigTypes := func(fd *ast.FuncDecl) {
		if fd == nil || fd.Type == nil {
			return
		}
		if fd.Recv != nil {
			for _, r := range fd.Recv.List {
				harvestTypes(r.Type)
			}
		}
		if fd.Type.Params != nil {
			for _, p := range fd.Type.Params.List {
				harvestTypes(p.Type)
			}
		}
		if fd.Type.Results != nil {
			for _, r := range fd.Type.Results.List {
				harvestTypes(r.Type)
			}
		}
	}

	// ---------- Initial roots ----------
	work := []*types.Func{}
	pushFunc := func(fn *types.Func) {
		if fn == nil || reachFunc[fn] {
			return
		}
		reachFunc[fn] = true
		work = append(work, fn)
	}
	// Root: main
	if mainFunc != nil {
		if fn, ok := info.Defs[mainFunc.Name].(*types.Func); ok {
			pushFunc(fn)
		}
	}
	// Root: top-level "var X = someFunc" or "const" init used as func idents
	for obj, decl := range funcDeclByObj {
		_ = decl
		_ = obj
	}
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || (gd.Tok != token.VAR && gd.Tok != token.CONST) {
			continue
		}
		for _, sp := range gd.Specs {
			vs := sp.(*ast.ValueSpec)
			for _, val := range vs.Values {
				if id, ok := val.(*ast.Ident); ok {
					if o := info.Uses[id]; o != nil {
						if fn, ok := o.(*types.Func); ok {
							pushFunc(fn)
							// also keep the variable being assigned (name spec)
							reachValue[vs] = true
						}
					}
				}
			}
		}
	}

	// ---------- BFS over reachable functions/methods ----------
	visitFuncBody := func(fd *ast.FuncDecl) {
		if fd == nil || fd.Body == nil {
			return
		}
		harvestFuncSigTypes(fd)
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				switch cal := x.Fun.(type) {
				case *ast.Ident:
					if obj := info.Uses[cal]; obj != nil {
						if fn, ok := obj.(*types.Func); ok {
							pushFunc(fn)
						}
					}
				case *ast.SelectorExpr:
					// method value or method expr
					if sel, ok := info.Selections[cal]; ok {
						if f, ok := sel.Obj().(*types.Func); ok {
							pushFunc(f)
						}
					} else {
						// package-qualified func
						if obj := info.Uses[cal.Sel]; obj != nil {
							if f, ok := obj.(*types.Func); ok {
								pushFunc(f)
							}
						}
					}
				}
				// harvest type args in builtins/conversions
				for _, a := range x.Args {
					harvestTypes(a)
				}
			case *ast.CompositeLit:
				harvestTypes(x.Type)
			case *ast.TypeAssertExpr:
				harvestTypes(x.Type)
			case *ast.ValueSpec:
				harvestTypes(x.Type)
				// Any referenced global var/const names counted as used
				for _, nm := range x.Names {
					if vs, ok := valueNameToSpec[nm.Name]; ok {
						reachValue[vs] = true
					}
				}
			case *ast.Ident:
				if vs, ok := valueNameToSpec[x.Name]; ok {
					reachValue[vs] = true
				}
			case *ast.UnaryExpr:
				if cl, ok := x.X.(*ast.CompositeLit); ok {
					harvestTypes(cl.Type)
				}
			}
			return true
		})
	}

	for len(work) > 0 {
		fn := work[0]
		work = work[1:]
		if fd := funcDeclByObj[fn]; fd != nil {
			visitFuncBody(fd)
		}
	}

	// ---------- Propagate type dependencies (types referenced inside types) ----------
	propagateTypes := func() {
		changed := true
		for changed {
			changed = false
			for tn := range reachType {
				if ts, ok := typeDeclByObj[tn]; ok && ts != nil {
					before := len(reachType)
					harvestTypes(ts.Type)
					if len(reachType) != before {
						changed = true
					}
				}
			}
		}
	}
	propagateTypes()

	// ---------- Interface → concrete method retention ----------
	// If a reachable named type (or *T) implements a reachable interface I,
	// then every method required by I on that receiver must be retained.
	for it := range reachIface {
		if it == nil {
			continue
		}
		// Interface's method names
		ifaceM := map[string]struct{}{}
		for i := 0; i < it.NumMethods(); i++ {
			ifaceM[it.Method(i).Name()] = struct{}{}
		}
		for nm := range reachNamed {
			// T and *T
			check := func(typ types.Type) {
				if !types.Implements(typ, it) {
					return
				}
				ms := types.NewMethodSet(typ)
				for i := 0; i < ms.Len(); i++ {
					sel := ms.At(i)
					if _, ok := ifaceM[sel.Obj().Name()]; !ok {
						continue
					}
					if mf, ok := sel.Obj().(*types.Func); ok && !reachFunc[mf] {
						pushFunc(mf)
					}
				}
			}
			check(nm)
			check(types.NewPointer(nm))
		}
	}
	// Drain any new funcs added by the interface bridge
	for len(work) > 0 {
		fn := work[0]
		work = work[1:]
		if fd := funcDeclByObj[fn]; fd != nil {
			visitFuncBody(fd)
		}
	}
	// A final type propagation in case method signatures pulled new types.
	propagateTypes()

	// ---------- Build filtered decl list ----------
	var kept []ast.Decl

	keepFuncDecl := func(fd *ast.FuncDecl) bool {
		if fd == nil {
			return false
		}
		if fd.Name.Name == "main" {
			return true
		}
		if obj, ok := info.Defs[fd.Name].(*types.Func); ok && obj != nil {
			return reachFunc[obj]
		}
		// If we can't resolve (unlikely), keep conservatively.
		return true
	}

	keepTypeSpec := func(ts *ast.TypeSpec) bool {
		if tn, ok := info.Defs[ts.Name].(*types.TypeName); ok && tn != nil {
			return reachType[tn]
		}
		return false
	}

	keepValueSpec := func(vs *ast.ValueSpec) bool {
		return reachValue[vs]
	}

	for _, d := range file.Decls {
		switch dd := d.(type) {
		case *ast.FuncDecl:
			if keepFuncDecl(dd) {
				kept = append(kept, d)
			}
		case *ast.GenDecl:
			switch dd.Tok {
			case token.TYPE:
				var specs []ast.Spec
				for _, sp := range dd.Specs {
					ts := sp.(*ast.TypeSpec)
					if keepTypeSpec(ts) {
						specs = append(specs, sp)
					}
				}
				if len(specs) > 0 {
					nd := *dd
					nd.Specs = specs
					kept = append(kept, &nd)
				}
			case token.VAR, token.CONST:
				var specs []ast.Spec
				for _, sp := range dd.Specs {
					vs := sp.(*ast.ValueSpec)
					if keepValueSpec(vs) {
						specs = append(specs, sp)
					}
				}
				if len(specs) > 0 {
					nd := *dd
					nd.Specs = specs
					kept = append(kept, &nd)
				}
			case token.IMPORT:
				// keep imports for now; go/format + go/types later can trim with 'sanitizeCode'
				kept = append(kept, d)
			default:
				kept = append(kept, d)
			}
		default:
			kept = append(kept, d)
		}
	}
	file.Decls = kept

	// Drop all comments (optional; matches your previous pass)
	file.Comments = nil
	for _, d := range file.Decls {
		if gd, ok := d.(*ast.GenDecl); ok {
			gd.Doc = nil
		}
		if fd, ok := d.(*ast.FuncDecl); ok {
			fd.Doc = nil
		}
	}

	// ---------- Stable-ish emission: keep package/imports + main first ----------
	var out bytes.Buffer
	_ = printer.Fprint(&out, fset, file)
	pretty, err := format.Source(out.Bytes())
	if err == nil {
		return pretty
	}
	return out.Bytes()
}
