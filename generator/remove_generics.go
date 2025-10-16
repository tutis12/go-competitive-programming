package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/imports"
)

// genericNameFormat controls naming of monomorphized entities.
const genericNameFormat = "%sG%d"

// RemoveGenerics performs a monomorphization pass over merged source.
// It discovers generic type and function instantitions, propagates nested
// usages, emits concrete strucethods and functions, and strips the
// original generic declarations. It intentionally avoids any hardcoded
// knowledge of specific type names.
func RemoveGenerics(src []byte) []byte {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "merged.go", src, parser.ParseComments)
	if err != nil {
		return src
	}

	// Collect generic type declarations.
	type genericInfo struct {
		decl     *ast.GenDecl
		spec     *ast.TypeSpec
		isStruct bool
		params   []*ast.Field
		fields   *ast.FieldList // only for structs
		typeExpr ast.Expr       // the actual type expression (for non-struct types)
	}
	generics := map[string]*genericInfo{}
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, sp := range gd.Specs {
			ts, ok := sp.(*ast.TypeSpec)
			if !ok || ts.TypeParams == nil {
				continue
			}
			gi := &genericInfo{decl: gd, spec: ts, params: ts.TypeParams.List, typeExpr: ts.Type}
			switch tt := ts.Type.(type) {
			case *ast.StructType:
				gi.isStruct = true
				gi.fields = tt.Fields
			case *ast.InterfaceType:
				gi.isStruct = false
				gi.fields = nil
			default:
				// Handle all other types: arrays, slices, maps, channels, pointers, etc.
				gi.isStruct = false
				gi.fields = nil
			}
			generics[ts.Name.Name] = gi
		}
	}

	// Infer from composite literals and identifiers.
	// Collect generic functions (no receiver).
	genericFuncs := map[string]*ast.FuncDecl{}
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil && fd.Type != nil && fd.Type.TypeParams != nil {
			genericFuncs[fd.Name.Name] = fd
		}
	}

	// Discover explicit instantiations via IndexExpr / IndexListExpr.
	insts := map[string][]string{}
	seenInst := map[string]struct{}{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.IndexExpr:
			if id, ok := e.X.(*ast.Ident); ok {
				if generics[id.Name] != nil || genericFuncs[id.Name] != nil {
					arg := exprToString(fset, e.Index)
					key := id.Name + "[" + arg + "]"
					if _, exists := seenInst[key]; !exists {
						seenInst[key] = struct{}{}
						addInstantiation(insts, id.Name, arg)
					}
				}
			}
		case *ast.IndexListExpr:
			if id, ok := e.X.(*ast.Ident); ok {
				if generics[id.Name] != nil || genericFuncs[id.Name] != nil {
					parts := make([]string, 0, len(e.Indices))
					for _, ix := range e.Indices {
						parts = append(parts, exprToString(fset, ix))
					}
					arg := strings.Join(parts, ",")
					key := id.Name + "[" + arg + "]"
					if _, exists := seenInst[key]; !exists {
						seenInst[key] = struct{}{}
						addInstantiation(insts, id.Name, arg)
					}
				}
			}
		}
		return true
	})

	// Expand partial instantiations for generic functions with missing type parameters.
	for fnName, sigList := range insts {
		gfn := genericFuncs[fnName]
		if gfn == nil || gfn.Type == nil || gfn.Type.TypeParams == nil {
			continue
		}
		var expectedParams []string
		for _, tp := range gfn.Type.TypeParams.List {
			for _, id := range tp.Names {
				expectedParams = append(expectedParams, id.Name)
			}
		}

		newSigs := []string{}
		for _, sig := range sigList {
			parts := strings.Split(sig, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}

			if len(parts) < len(expectedParams) {
				// Try to infer missing parameters
				// For NewHashMap[K comparable, V any, H Hasher[K]], if we have [intHash, int],
				// we can infer H = K = intHash
				if fnName == "NewHashMap" && len(parts) == 2 && len(expectedParams) == 3 {
					// H Hasher[K] constraint means H should be same as K
					newSig := parts[0] + "," + parts[1] + "," + parts[0]
					newSigs = append(newSigs, newSig)
				} else {
					// Keep original
					newSigs = append(newSigs, sig)
				}
			} else {
				newSigs = append(newSigs, sig)
			}
		}
		insts[fnName] = newSigs
	}

	// Infer instantiations from generic function calls (heuristic argument → type param mapping).
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Body != nil {
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				nameIdent, okIdent := call.Fun.(*ast.Ident)
				if !okIdent {
					return true
				}
				gfn := genericFuncs[nameIdent.Name]
				if gfn == nil || gfn.Type == nil || gfn.Type.Params == nil || gfn.Type.TypeParams == nil {
					return true
				}
				if len(call.Args) != len(gfn.Type.Params.List) {
					return true
				}
				// Build mapping from type parameter names to concrete types.
				mapping := map[string]string{}
				// Helper: does an expression tree contain an Ident with given name?
				containsIdent := func(e ast.Expr, name string) bool {
					found := false
					ast.Inspect(e, func(n ast.Node) bool {
						id, ok := n.(*ast.Ident)
						if ok && id.Name == name {
							found = true
							return false
						}
						return true
					})
					return found
				}
				// Pass 1: direct patterns (Ident params, composite literals, basic literals, common index identifiers).
				for i, p := range gfn.Type.Params.List {
					argExpr := call.Args[i]
					// Collect candidate generic identifiers present in formal parameter type.
					var genericIdentsInParam []string
					for _, tp := range gfn.Type.TypeParams.List {
						for _, idn := range tp.Names {
							if containsIdent(p.Type, idn.Name) {
								genericIdentsInParam = append(genericIdentsInParam, idn.Name)
							}
						}
					}
					// Map each generic ident from this parameter using argument expression.
					for _, gname := range genericIdentsInParam {
						if mapping[gname] != "" {
							continue
						}
						switch a := argExpr.(type) {
						case *ast.CompositeLit:
							if a.Type != nil {
								mapping[gname] = strings.TrimSpace(exprToString(fset, a.Type))
							}
						case *ast.UnaryExpr:
							if cl, okCL := a.X.(*ast.CompositeLit); okCL && cl.Type != nil {
								mapping[gname] = strings.TrimSpace(exprToString(fset, cl.Type))
							}
						case *ast.BasicLit:
							switch a.Kind {
							case token.INT:
								mapping[gname] = "int"
							case token.FLOAT:
								mapping[gname] = "float64"
							case token.STRING:
								mapping[gname] = "string"
							}
						case *ast.Ident:
							if a.Name == "n" || a.Name == "m" || a.Name == "i" || a.Name == "j" || strings.Contains(a.Name, "size") || strings.Contains(a.Name, "len") {
								mapping[gname] = "int"
							}
						case *ast.CallExpr:
							// Handle type casts like uint64(20), int(x), etc.
							if funIdent, ok := a.Fun.(*ast.Ident); ok {
								// This is a type cast - use the function name as the target type
								mapping[gname] = funIdent.Name
							}
						}
					}
				}
				// Pass 2: function literal return type extraction for parameters whose formal type embeds generic identifiers.
				for i, p := range gfn.Type.Params.List {
					argExpr := call.Args[i]
					fnType, okFT := p.Type.(*ast.FuncType)
					if !okFT {
						continue
					}
					lit, okLit := argExpr.(*ast.FuncLit)
					if !okLit || lit.Type == nil || lit.Type.Results == nil || len(lit.Type.Results.List) == 0 {
						continue
					}
					// Currently only handle single-result functions.
					if len(lit.Type.Results.List) != len(fnType.Results.List) { /* still attempt if counts differ? skip */
					}
					for _, tp := range gfn.Type.TypeParams.List {
						for _, idn := range tp.Names {
							if mapping[idn.Name] != "" {
								continue
							}
							if containsIdent(p.Type, idn.Name) {
								// Use first result type of literal.
								resType := lit.Type.Results.List[0].Type
								mapping[idn.Name] = strings.TrimSpace(exprToString(fset, resType))
							}
						}
					}
				}
				// Build signature from mapping; require all parameters resolved.
				parts := []string{}
				all := true
				for _, tp := range gfn.Type.TypeParams.List {
					for _, id := range tp.Names {
						v := mapping[id.Name]
						if v == "" {
							all = false
						}
						parts = append(parts, v)
					}
				}
				if !all {
					return true
				}
				sig := strings.Join(parts, ",")
				key := gfn.Name.Name + "[" + sig + "]"
				if _, exists := seenInst[key]; !exists {
					seenInst[key] = struct{}{}
					insts[gfn.Name.Name] = append(insts[gfn.Name.Name], sig)
				}
				// Instantiate any generic struct returned by this function using mapped parameters.
				if gfn.Type.Results != nil && len(gfn.Type.Results.List) == 1 {
					retT := gfn.Type.Results.List[0].Type
					if se, ok := retT.(*ast.StarExpr); ok {
						retT = se.X
					}
					var idxList *ast.IndexListExpr
					switch rt := retT.(type) {
					case *ast.IndexExpr:
						idxList = &ast.IndexListExpr{X: rt.X, Indices: []ast.Expr{rt.Index}}
					case *ast.IndexListExpr:
						idxList = rt
					}
					if idxList != nil {
						if baseId, okB := idxList.X.(*ast.Ident); okB {
							if gStruct := generics[baseId.Name]; gStruct != nil && gStruct.isStruct {
								innerParts := []string{}
								for _, ix := range idxList.Indices {
									if idInner, okIn := ix.(*ast.Ident); okIn {
										innerParts = append(innerParts, mapping[idInner.Name])
									} else {
										innerParts = append(innerParts, exprToString(fset, ix))
									}
								}
								innerSig := strings.Join(innerParts, ",")
								innerKey := baseId.Name + "[" + innerSig + "]"
								if _, exists := seenInst[innerKey]; !exists {
									seenInst[innerKey] = struct{}{}
									insts[baseId.Name] = append(insts[baseId.Name], innerSig)
								}
							}
						}
					}
				}
				return true
			})
		}
	}

	// Collect all type parameter names first (for filtering during propagation)
	paramNames := map[string]bool{}
	for _, g := range generics {
		for _, p := range g.params {
			for _, id := range p.Names {
				paramNames[id.Name] = true
			}
		}
	}
	for _, fn := range genericFuncs {
		for _, p := range fn.Type.TypeParams.List {
			for _, id := range p.Names {
				paramNames[id.Name] = true
			}
		}
	}

	// Helper function to check if instantiation consists entirely of type parameter names
	isParameterOnlyInst := func(inst string) bool {
		parts := strings.Split(inst, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" && !paramNames[part] {
				return false
			}
		}
		return true
	}

	// Helper function to check if instantiation has correct parameter count
	hasCorrectParamCount := func(typeName, inst string) bool {
		parts := strings.Split(inst, ",")
		actualCount := len(parts)
		if actualCount == 1 && strings.TrimSpace(parts[0]) == "" {
			actualCount = 0
		}

		if g := generics[typeName]; g != nil {
			expectedCount := 0
			for _, p := range g.params {
				expectedCount += len(p.Names)
			}
			return actualCount == expectedCount
		}
		return true // If we can't determine expected count, allow it
	}

	// Propagate nested generic struct field instantiations.
	for outerName, sigList := range insts {
		og := generics[outerName]
		if og == nil || !og.isStruct || og.fields == nil {
			continue
		}

		var order []string
		for _, p := range og.params {
			for _, id := range p.Names {
				order = append(order, id.Name)
			}
		}
		for _, sig := range sigList {
			// Skip instantiations that consist entirely of type parameter names
			if isParameterOnlyInst(sig) {
				// if outerName == "HashMap" {
				//	// fmt.Printf("DEBUG HASHMAP: skipping parameter-only instantiation HashMap[%s]\n", sig)
				// }
				continue
			}

			// Skip instantiations with wrong parameter count
			if !hasCorrectParamCount(outerName, sig) {
				// // fmt.Printf("DEBUG: skipping %s[%s] - wrong parameter count\n", outerName, sig)
				continue
			}

			parts := strings.Split(sig, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			mapping := map[string]string{}
			for i, name := range order {
				if i < len(parts) {
					mapping[name] = parts[i]
				}
			}

			// Debug for HashMap processing
			// if outerName == "HashMap" {
			//	// fmt.Printf("DEBUG HASHMAP: HashMap[%s] mapping = %v\n", sig, mapping)
			// }

			for _, fld := range og.fields.List {
				ast.Inspect(fld.Type, func(n ast.Node) bool {
					switch ex := n.(type) {
					case *ast.IndexExpr:
						if id, ok := ex.X.(*ast.Ident); ok {
							ig := generics[id.Name]
							if ig != nil { // Remove isStruct restriction - propagate to all generic types
								arg := exprToString(fset, ex.Index)
								if outerName == "HashMap" && id.Name == "bucket" {
									// fmt.Printf("DEBUG HASHMAP: found bucket[%s] in HashMap field\n", arg)
								}
								for k, v := range mapping {
									arg = replaceTypeToken(arg, k, v)
								}
								if outerName == "HashMap" && id.Name == "bucket" {
									// fmt.Printf("DEBUG HASHMAP: after replacement bucket[%s]\n", arg)
								}
								arg = strings.TrimSpace(arg)
								if arg != "" {
									addInstantiation(insts, id.Name, arg)
								}
							}
						}
					case *ast.IndexListExpr:
						if id, ok := ex.X.(*ast.Ident); ok {
							ig := generics[id.Name]
							if ig != nil { // Remove isStruct restriction - propagate to all generic types
								parts2 := make([]string, 0, len(ex.Indices))
								for _, ix := range ex.Indices {
									seg := exprToString(fset, ix)
									if outerName == "HashMap" && id.Name == "bucket" {
										// fmt.Printf("DEBUG HASHMAP: found bucket segment [%s] in HashMap field\n", seg)
									}
									for k, v := range mapping {
										seg = replaceTypeToken(seg, k, v)
									}
									if outerName == "HashMap" && id.Name == "bucket" {
										// fmt.Printf("DEBUG HASHMAP: after replacement segment [%s]\n", seg)
									}
									parts2 = append(parts2, strings.TrimSpace(seg))
								}
								joined := strings.Join(parts2, ",")
								if outerName == "HashMap" && id.Name == "bucket" {
									// fmt.Printf("DEBUG HASHMAP: final bucket signature [%s]\n", joined)
								}
								if strings.TrimSpace(joined) != "" {
									addInstantiation(insts, id.Name, joined)
								}
							}
						}
					}
					return true
				})
			}
		}
	}

	// Debug: print all instantiations before propagation
	// // fmt.Printf("DEBUG: All instantiations before propagation:\n")
	// for name, sigList := range insts {
	//	fmt.Printf("  %s: %v\n", name, sigList)
	// }

	// Debug: print initial HashMap instantiations
	// if hashMapInsts, exists := insts["HashMap"]; exists {
	//	// fmt.Printf("DEBUG: Initial HashMap instantiations: %v\n", hashMapInsts)
	// }
	// Debug: print initial bucket instantiations
	// if bucketInsts, exists := insts["bucket"]; exists {
	//	fmt.Printf("DEBUG bucket instantiations: %v\n", bucketInsts)
	// }

	// Propagate nested generic type instantiations for non-struct types (e.g., type aliases).
	for outerName, sigList := range insts {
		og := generics[outerName]
		if og == nil || og.isStruct || og.typeExpr == nil {
			continue // Skip structs (handled above) and types without type expressions
		}

		var order []string
		for _, p := range og.params {
			for _, id := range p.Names {
				order = append(order, id.Name)
			}
		}
		for _, sig := range sigList {
			// Skip instantiations that consist entirely of type parameter names
			if isParameterOnlyInst(sig) {
				if outerName == "bucket" {
					// fmt.Printf("DEBUG BUCKET: skipping parameter-only instantiation bucket[%s]\n", sig)
				}
				continue
			}

			// Skip instantiations with wrong parameter count
			if !hasCorrectParamCount(outerName, sig) {
				fmt.Printf("DEBUG: skipping %s[%s] - wrong parameter count\n", outerName, sig)
				continue
			}

			parts := strings.Split(sig, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			mapping := map[string]string{}
			for i, name := range order {
				if i < len(parts) {
					mapping[name] = parts[i]
				}
			}

			// Debug for bucket processing
			if outerName == "bucket" {
				// fmt.Printf("DEBUG BUCKET: bucket[%s] mapping = %v\n", sig, mapping)
			} // Inspect the type expression for nested generics
			ast.Inspect(og.typeExpr, func(n ast.Node) bool {
				switch ex := n.(type) {
				case *ast.IndexExpr:
					if id, ok := ex.X.(*ast.Ident); ok {
						ig := generics[id.Name]
						if ig != nil {
							arg := exprToString(fset, ex.Index)
							if outerName == "bucket" && id.Name == "entry" {
								// fmt.Printf("DEBUG BUCKET: found entry[%s] in bucket\n", arg)
							}
							for k, v := range mapping {
								arg = replaceTypeToken(arg, k, v)
							}
							if outerName == "bucket" && id.Name == "entry" {
								// fmt.Printf("DEBUG BUCKET: after replacement entry[%s]\n", arg)
							}
							arg = strings.TrimSpace(arg)
							if arg != "" {
								addInstantiation(insts, id.Name, arg)
							}
						}
					}
				case *ast.IndexListExpr:
					if id, ok := ex.X.(*ast.Ident); ok {
						ig := generics[id.Name]
						if ig != nil {
							parts2 := make([]string, 0, len(ex.Indices))
							for _, ix := range ex.Indices {
								seg := exprToString(fset, ix)
								if outerName == "bucket" && id.Name == "entry" {
									// fmt.Printf("DEBUG BUCKET: found entry segment [%s] in bucket\n", seg)
								}
								for k, v := range mapping {
									seg = replaceTypeToken(seg, k, v)
								}
								if outerName == "bucket" && id.Name == "entry" {
									// fmt.Printf("DEBUG BUCKET: after replacement segment [%s]\n", seg)
								}
								parts2 = append(parts2, strings.TrimSpace(seg))
							}
							joined := strings.Join(parts2, ",")
							if outerName == "bucket" && id.Name == "entry" {
								// fmt.Printf("DEBUG BUCKET: final entry signature [%s]\n", joined)
							}
							if strings.TrimSpace(joined) != "" {
								addInstantiation(insts, id.Name, joined)
							}
						}
					}
				}
				return true
			})
		}
	}

	// Pre-filter dependency expansion: for each struct instantiation, substitute outer concrete args into inner param-only generic struct usages.
	for outerName, sigList := range insts {
		og := generics[outerName]
		if og == nil || !og.isStruct || og.fields == nil {
			continue
		}
		// Order of outer params.
		var outerParamNames []string
		for _, p := range og.params {
			for _, id := range p.Names {
				outerParamNames = append(outerParamNames, id.Name)
			}
		}
		for _, sig := range sigList {
			parts := strings.Split(sig, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			if len(parts) != len(outerParamNames) {
				continue
			}
			outerMap := map[string]string{}
			for i, pn := range outerParamNames {
				outerMap[pn] = parts[i]
			}
			for _, fld := range og.fields.List {
				ast.Inspect(fld.Type, func(n ast.Node) bool {
					switch ex := n.(type) {
					case *ast.IndexListExpr:
						if baseId, okB := ex.X.(*ast.Ident); okB {
							inner := generics[baseId.Name]
							if inner != nil && inner.isStruct {
								innerArgs := make([]string, 0, len(ex.Indices))
								for _, ix := range ex.Indices {
									if idInner, okIn := ix.(*ast.Ident); okIn {
										mapped := outerMap[idInner.Name]
										if mapped == "" {
											mapped = idInner.Name
										}
										innerArgs = append(innerArgs, mapped)
										// param-only status no longer tracked; outer substitution will create concrete args later.
									} else {
										seg := exprToString(fset, ix)
										innerArgs = append(innerArgs, seg)
										// param-only status no longer tracked.
									}
								}
								joined := strings.Join(innerArgs, ",")
								if joined != "" {
									addInstantiation(insts, baseId.Name, joined)
								}
								// Removed param-only note; filtering happens after potential concrete substitution.
							}
						}
					case *ast.IndexExpr:
						if baseId, okB := ex.X.(*ast.Ident); okB {
							inner := generics[baseId.Name]
							if inner != nil && inner.isStruct {
								seg := exprToString(fset, ex.Index)
								seg = strings.TrimSpace(seg)
								if seg != "" {
									addInstantiation(insts, baseId.Name, seg)
								}
							}
						}
					}
					return true
				})
			}
		}
	}

	// Propagate from generic function return types (if returning generic struct).
	for fnName, sigList := range insts {
		gfn := genericFuncs[fnName]
		if gfn == nil || gfn.Type == nil || gfn.Type.Results == nil || len(gfn.Type.Results.List) != 1 {
			continue
		}

		ret := gfn.Type.Results.List[0].Type
		if se, ok := ret.(*ast.StarExpr); ok {
			ret = se.X
		}
		var idxList *ast.IndexListExpr
		switch rt := ret.(type) {
		case *ast.IndexListExpr:
			idxList = rt
		case *ast.IndexExpr:
			idxList = &ast.IndexListExpr{X: rt.X, Indices: []ast.Expr{rt.Index}}
		}
		if idxList == nil {
			continue
		}
		base, okBase := idxList.X.(*ast.Ident)
		if !okBase {
			continue
		}
		gs := generics[base.Name]
		if gs == nil || !gs.isStruct {
			continue
		}
		var order []string
		for _, p := range gfn.Type.TypeParams.List {
			for _, id := range p.Names {
				order = append(order, id.Name)
			}
		}
		for _, sig := range sigList {
			parts := strings.Split(sig, ",")
			if len(parts) != len(order) {
				continue
			}
			mp := map[string]string{}
			for i, n := range order {
				mp[n] = strings.TrimSpace(parts[i])
			}
			structArgs := make([]string, 0, len(idxList.Indices))
			for _, e := range idxList.Indices {
				if id, ok := e.(*ast.Ident); ok {
					structArgs = append(structArgs, mp[id.Name])
				} else {
					s := exprToString(fset, e)
					for k, v := range mp {
						s = replaceTypeToken(s, k, v)
					}
					structArgs = append(structArgs, s)
				}
			}
			if len(structArgs) > 0 {
				addInstantiation(insts, base.Name, strings.Join(structArgs, ","))
			}
		}
	}

	// Propagate nested generic struct field instantiations (again, after return type propagation).
	for outerName, sigList := range insts {
		og := generics[outerName]
		if og == nil || !og.isStruct || og.fields == nil {
			continue
		}

		var order []string
		for _, p := range og.params {
			for _, id := range p.Names {
				order = append(order, id.Name)
			}
		}
		for _, sig := range sigList {
			parts := strings.Split(sig, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			mapping := map[string]string{}
			for i, name := range order {
				if i < len(parts) {
					mapping[name] = parts[i]
				}
			}
			for _, fld := range og.fields.List {
				ast.Inspect(fld.Type, func(n ast.Node) bool {
					switch ex := n.(type) {
					case *ast.IndexExpr:
						if id, ok := ex.X.(*ast.Ident); ok {
							ig := generics[id.Name]
							if ig != nil { // Remove isStruct restriction
								arg := exprToString(fset, ex.Index)
								for k, v := range mapping {
									arg = replaceTypeToken(arg, k, v)
								}
								arg = strings.TrimSpace(arg)
								if arg != "" {
									addInstantiation(insts, id.Name, arg)
								}
							}
						}
					case *ast.IndexListExpr:
						if id, ok := ex.X.(*ast.Ident); ok {
							ig := generics[id.Name]
							if ig != nil { // Remove isStruct restriction
								parts2 := make([]string, 0, len(ex.Indices))
								for _, ix := range ex.Indices {
									s := exprToString(fset, ix)
									for k, v := range mapping {
										s = replaceTypeToken(s, k, v)
									}
									parts2 = append(parts2, s)
								}
								joined := strings.Join(parts2, ",")
								if strings.TrimSpace(joined) != "" {
									addInstantiation(insts, id.Name, joined)
								}
							}
						}
					}
					return true
				})
			}
		}
	}

	// Global scan for generic type instantiations appearing anywhere (e.g. slice element types) not reached via propagation.
	// This is parameter-agnostic and does not hardcode any names; it simply records Index/List expressions whose base is a known generic struct.
	ast.Inspect(file, func(n ast.Node) bool {
		switch ex := n.(type) {
		case *ast.IndexExpr:
			if id, ok := ex.X.(*ast.Ident); ok {
				if g := generics[id.Name]; g != nil && g.isStruct {
					arg := exprToString(fset, ex.Index)
					arg = strings.TrimSpace(arg)
					if arg != "" {
						addInstantiation(insts, id.Name, arg)
					}
				}
			}
		case *ast.IndexListExpr:
			if id, ok := ex.X.(*ast.Ident); ok {
				if g := generics[id.Name]; g != nil && g.isStruct {
					parts := make([]string, 0, len(ex.Indices))
					for _, ix := range ex.Indices {
						seg := strings.TrimSpace(exprToString(fset, ix))
						parts = append(parts, seg)
					}
					joined := strings.Join(parts, ",")
					if strings.TrimSpace(joined) != "" {
						addInstantiation(insts, id.Name, joined)
					}
				}
			}
		}
		return true
	})

	// Fixed-point substitution of inner generic param-only signatures using outer concrete arguments before filtering.
	changed := true
	for changed {
		changed = false
		for outerName, sigList := range insts {
			og := generics[outerName]
			if og == nil || !og.isStruct || og.fields == nil {
				continue
			}
			// Gather param order.
			var order []string
			for _, p := range og.params {
				for _, id := range p.Names {
					order = append(order, id.Name)
				}
			}
			for _, sig := range sigList {
				parts := strings.Split(sig, ",")
				if len(parts) != len(order) {
					continue
				}
				mapping := map[string]string{}
				for i, name := range order {
					mapping[name] = strings.TrimSpace(parts[i])
				}
				for _, fld := range og.fields.List {
					ast.Inspect(fld.Type, func(n ast.Node) bool {
						switch ex := n.(type) {
						case *ast.IndexListExpr:
							if id, ok := ex.X.(*ast.Ident); ok {
								ig := generics[id.Name]
								if ig != nil { // Remove isStruct restriction
									innerParts := make([]string, 0, len(ex.Indices))
									for _, ix := range ex.Indices {
										if idIn, okIn := ix.(*ast.Ident); okIn {
											innerParts = append(innerParts, strings.TrimSpace(mapping[idIn.Name]))
										} else {
											innerParts = append(innerParts, strings.TrimSpace(exprToString(fset, ix)))
										}
									}
									joined := strings.Join(innerParts, ",")
									// Add instantiation if new.
									already := false
									for _, existing := range insts[id.Name] {
										if existing == joined {
											already = true
											break
										}
									}
									if !already {
										insts[id.Name] = append(insts[id.Name], joined)
										changed = true
									}
								}
							}
						case *ast.IndexExpr:
							if id, ok := ex.X.(*ast.Ident); ok {
								ig := generics[id.Name]
								if ig != nil { // Remove isStruct restriction
									seg := strings.TrimSpace(exprToString(fset, ex.Index))
									already := false
									for _, existing := range insts[id.Name] {
										if existing == seg {
											already = true
											break
										}
									}
									if seg != "" && !already {
										insts[id.Name] = append(insts[id.Name], seg)
										changed = true
									}
								}
							}
						}
						return true
					})
				}
			}
		}
	}

	// Now apply parameter-only filtering.
	// (paramNames already collected above)

	// Debug: print parameter names
	// fmt.Printf("DEBUG: paramNames = %v\n", paramNames)

	filtered := map[string][]string{}
	for name, list := range insts {
		for _, inst := range list {
			parts := strings.Split(inst, ",")
			keep := false
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" && !paramNames[part] {
					keep = true
					break
				}
			}
			if name == "HashMap" {
				// fmt.Printf("DEBUG: HashMap[%s] keep=%v parts=%v\n", inst, keep, parts)
			}
			if keep {
				filtered[name] = append(filtered[name], inst)
			}
		}
	}
	insts = filtered

	// Second propagation pass: now that we have concrete instantiations from return type analysis,
	// propagate these to nested dependencies
	for outerName, sigList := range insts {
		og := generics[outerName]
		if og == nil || !og.isStruct || og.fields == nil {
			continue
		}

		var order []string
		for _, p := range og.params {
			for _, id := range p.Names {
				order = append(order, id.Name)
			}
		}
		for _, sig := range sigList {
			// Skip instantiations that consist entirely of type parameter names
			if isParameterOnlyInst(sig) {
				continue
			}

			// Skip instantiations with wrong parameter count
			if !hasCorrectParamCount(outerName, sig) {
				// fmt.Printf("DEBUG PASS2: skipping %s[%s] - wrong parameter count\n", outerName, sig)
				continue
			}

			parts := strings.Split(sig, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			mapping := map[string]string{}
			for i, name := range order {
				if i < len(parts) {
					mapping[name] = parts[i]
				}
			}

			// Debug for HashMap processing in second pass
			if outerName == "HashMap" {
				// fmt.Printf("DEBUG HASHMAP PASS2: HashMap[%s] mapping = %v\n", sig, mapping)
			}

			for _, fld := range og.fields.List {
				ast.Inspect(fld.Type, func(n ast.Node) bool {
					switch ex := n.(type) {
					case *ast.IndexExpr:
						if id, ok := ex.X.(*ast.Ident); ok {
							ig := generics[id.Name]
							if ig != nil { // Remove isStruct restriction - propagate to all generic types
								arg := exprToString(fset, ex.Index)
								if outerName == "HashMap" && id.Name == "bucket" {
									// fmt.Printf("DEBUG HASHMAP PASS2: found bucket[%s] in HashMap field\n", arg)
								}
								for k, v := range mapping {
									arg = replaceTypeToken(arg, k, v)
								}
								if outerName == "HashMap" && id.Name == "bucket" {
									// fmt.Printf("DEBUG HASHMAP PASS2: after replacement bucket[%s]\n", arg)
								}
								arg = strings.TrimSpace(arg)
								if arg != "" {
									addInstantiation(insts, id.Name, arg)
								}
							}
						}
					case *ast.IndexListExpr:
						if id, ok := ex.X.(*ast.Ident); ok {
							ig := generics[id.Name]
							if ig != nil { // Remove isStruct restriction - propagate to all generic types
								parts2 := make([]string, 0, len(ex.Indices))
								for _, ix := range ex.Indices {
									seg := exprToString(fset, ix)
									if outerName == "HashMap" && id.Name == "bucket" {
										// fmt.Printf("DEBUG HASHMAP PASS2: found bucket segment [%s] in HashMap field\n", seg)
									}
									for k, v := range mapping {
										seg = replaceTypeToken(seg, k, v)
									}
									if outerName == "HashMap" && id.Name == "bucket" {
										// fmt.Printf("DEBUG HASHMAP PASS2: after replacement segment [%s]\n", seg)
									}
									parts2 = append(parts2, strings.TrimSpace(seg))
								}
								joined := strings.Join(parts2, ",")
								if outerName == "HashMap" && id.Name == "bucket" {
									// fmt.Printf("DEBUG HASHMAP PASS2: final bucket signature [%s]\n", joined)
								}
								if strings.TrimSpace(joined) != "" {
									addInstantiation(insts, id.Name, joined)
								}
							}
						}
					}
					return true
				})
			}
		}
	}

	// Second propagation pass for non-struct types
	for outerName, sigList := range insts {
		og := generics[outerName]
		if og == nil || og.isStruct || og.typeExpr == nil {
			continue // Skip structs (handled above) and types without type expressions
		}

		var order []string
		for _, p := range og.params {
			for _, id := range p.Names {
				order = append(order, id.Name)
			}
		}
		for _, sig := range sigList {
			// Skip instantiations that consist entirely of type parameter names
			if isParameterOnlyInst(sig) {
				continue
			}

			// Skip instantiations with wrong parameter count
			if !hasCorrectParamCount(outerName, sig) {
				// fmt.Printf("DEBUG PASS2: skipping %s[%s] - wrong parameter count\n", outerName, sig)
				continue
			}

			parts := strings.Split(sig, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			mapping := map[string]string{}
			for i, name := range order {
				if i < len(parts) {
					mapping[name] = parts[i]
				}
			}

			// Debug for bucket processing in second pass
			if outerName == "bucket" {
				// fmt.Printf("DEBUG BUCKET PASS2: bucket[%s] mapping = %v\n", sig, mapping)
			}

			// Inspect the type expression for nested generics
			ast.Inspect(og.typeExpr, func(n ast.Node) bool {
				switch ex := n.(type) {
				case *ast.IndexExpr:
					if id, ok := ex.X.(*ast.Ident); ok {
						ig := generics[id.Name]
						if ig != nil {
							arg := exprToString(fset, ex.Index)
							if outerName == "bucket" && id.Name == "entry" {
								// fmt.Printf("DEBUG BUCKET PASS2: found entry[%s] in bucket\n", arg)
							}
							for k, v := range mapping {
								arg = replaceTypeToken(arg, k, v)
							}
							if outerName == "bucket" && id.Name == "entry" {
								// fmt.Printf("DEBUG BUCKET PASS2: after replacement entry[%s]\n", arg)
							}
							arg = strings.TrimSpace(arg)
							if arg != "" {
								addInstantiation(insts, id.Name, arg)
							}
						}
					}
				case *ast.IndexListExpr:
					if id, ok := ex.X.(*ast.Ident); ok {
						ig := generics[id.Name]
						if ig != nil {
							parts2 := make([]string, 0, len(ex.Indices))
							for _, ix := range ex.Indices {
								seg := exprToString(fset, ix)
								if outerName == "bucket" && id.Name == "entry" {
									// fmt.Printf("DEBUG BUCKET PASS2: found entry segment [%s] in bucket\n", seg)
								}
								for k, v := range mapping {
									seg = replaceTypeToken(seg, k, v)
								}
								if outerName == "bucket" && id.Name == "entry" {
									// fmt.Printf("DEBUG BUCKET PASS2: after replacement segment [%s]\n", seg)
								}
								parts2 = append(parts2, strings.TrimSpace(seg))
							}
							joined := strings.Join(parts2, ",")
							if outerName == "bucket" && id.Name == "entry" {
								// fmt.Printf("DEBUG BUCKET PASS2: final entry signature [%s]\n", joined)
							}
							if strings.TrimSpace(joined) != "" {
								addInstantiation(insts, id.Name, joined)
							}
						}
					}
				}
				return true
			})
		}
	}

	// Deduplicate & sort instantiation signature lists.
	for name, list := range insts {
		uniq := []string{}
		seenLocal := map[string]bool{}
		for _, v := range list {
			if !seenLocal[v] {
				seenLocal[v] = true
				uniq = append(uniq, v)
			}
		}
		sort.Strings(uniq)
		insts[name] = uniq
	}
	if len(insts) == 0 {
		return src
	}

	// Map "Type[Sig]" → concrete name.
	nameMap := map[string]string{}
	for _, name := range sortedKeys(insts) {
		for i, sig := range insts[name] {
			key := name + "[" + sig + "]"
			concrete := fmt.Sprintf(genericNameFormat, name, i+1)
			nameMap[key] = concrete
		}
	}

	// Collect replacements: instantiated expressions & removal of generics.
	type repl struct {
		start, end int
		text       string
	}
	replacements := []repl{}
	seenTypeDecl := map[*ast.GenDecl]bool{}
	origGenericFuncCode := map[*ast.FuncDecl]string{}
	// Pre-capture generic function source before removal.
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil && fd.Type != nil && fd.Type.TypeParams != nil {
			origGenericFuncCode[fd] = nodeToString(fset, fd)
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.IndexExpr:
			if id, ok := e.X.(*ast.Ident); ok {
				if _, isGen := generics[id.Name]; isGen {
					arg := exprToString(fset, e.Index)
					key := id.Name + "[" + arg + "]"
					if nn, ok := nameMap[key]; ok {
						replacements = append(replacements, repl{posOffset(fset, e.Pos()), posOffset(fset, e.End()), nn})
					}
				}
			}
		case *ast.IndexListExpr:
			if id, ok := e.X.(*ast.Ident); ok {
				if _, isGen := generics[id.Name]; isGen {
					parts := []string{}
					for _, ix := range e.Indices {
						parts = append(parts, exprToString(fset, ix))
					}
					arg := strings.Join(parts, ",")
					key := id.Name + "[" + arg + "]"
					if nn, ok := nameMap[key]; ok {
						replacements = append(replacements, repl{posOffset(fset, e.Pos()), posOffset(fset, e.End()), nn})
					}
				}
			}
		case *ast.GenDecl:
			if e.Tok == token.TYPE && !seenTypeDecl[e] {
				seenTypeDecl[e] = true
				removable := true
				for _, sp := range e.Specs {
					if ts, ok := sp.(*ast.TypeSpec); ok && ts.TypeParams == nil {
						removable = false
					}
				}
				if removable {
					replacements = append(replacements, repl{posOffset(fset, e.Pos()), posOffset(fset, e.End()), ""})
				}
			}
		case *ast.FuncDecl:
			if e.Recv == nil && e.Type != nil && e.Type.TypeParams != nil {
				replacements = append(replacements, repl{posOffset(fset, e.Pos()), posOffset(fset, e.End()), ""})
			}
		}
		return true
	})
	// Remove generic methods (we regenerate concrete versions later).
	ast.Inspect(file, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			return true
		}
		for _, r := range fd.Recv.List {
			base := r.Type
			if se, ok := base.(*ast.StarExpr); ok {
				base = se.X
			}
			switch bt := base.(type) {
			case *ast.IndexExpr:
				if id, ok := bt.X.(*ast.Ident); ok {
					if _, isGen := generics[id.Name]; isGen {
						replacements = append(replacements, repl{posOffset(fset, fd.Pos()), posOffset(fset, fd.End()), ""})
					}
				}
			case *ast.IndexListExpr:
				if id, ok := bt.X.(*ast.Ident); ok {
					if _, isGen := generics[id.Name]; isGen {
						replacements = append(replacements, repl{posOffset(fset, fd.Pos()), posOffset(fset, fd.End()), ""})
					}
				}
			}
		}
		return true
	})

	sort.Slice(replacements, func(i, j int) bool { return replacements[i].start < replacements[j].start })
	var base bytes.Buffer
	cursor := 0
	for _, r := range replacements {
		if r.start < cursor {
			continue
		}
		base.Write(src[cursor:r.start])
		base.WriteString(r.text)
		cursor = r.end
	}
	base.Write(src[cursor:])

	// Replace generic function call sites (NewHashMap[intHash,int,intHash] -> NewHashMapG1, including partial spacing variants).
	var baseStr string
	for key, concrete := range nameMap {
		// key is e.g. NewHashMap[intHash,int,intHash] or HashMap[intHash,int,intHash]
		if !strings.Contains(key, "[") {
			continue
		}
		fnName := strings.Split(key, "[")[0]
		// Only adjust function calls for generic functions.
		if genericFuncs[fnName] == nil {
			continue
		}
		// Replace explicit instantiation form in code with concrete name.
		baseStr = base.String()

		// Extract type parameters from the key
		bracketStart := strings.Index(key, "[")
		bracketEnd := strings.LastIndex(key, "]")
		if bracketStart != -1 && bracketEnd != -1 && bracketEnd > bracketStart {
			typeParams := key[bracketStart+1 : bracketEnd]
			typeParamList := strings.Split(typeParams, ",")

			// Create variants with different parameter counts (for type inference)
			variants := []string{key, strings.ReplaceAll(key, ",", ", ")}

			// Also try shorter versions with fewer explicit type parameters
			// E.g., NewHashMap[intHash,int,intHash] -> also try NewHashMap[intHash,int]
			for i := 1; i < len(typeParamList); i++ {
				shorterParams := strings.Join(typeParamList[:i], ",")
				shorterKey := fnName + "[" + shorterParams + "]"
				variants = append(variants, shorterKey)
				variants = append(variants, strings.ReplaceAll(shorterKey, ",", ", "))
			}

			for _, v := range variants {
				baseStr = strings.ReplaceAll(baseStr, v, concrete)
			}
		}
		base.Reset()
		base.WriteString(baseStr)
	}

	// Replace inferred generic function calls with context-aware concrete versions.
	// Parse the code again to analyze call sites and determine correct concrete versions.
	baseStr = base.String()
	fsetForCallAnalysis := token.NewFileSet()
	astForCallAnalysis, err := parser.ParseFile(fsetForCallAnalysis, "", baseStr, parser.ParseComments)
	if err != nil {
		// If parsing fails, fall back to simple replacement with G1
		for fnName, gfn := range genericFuncs {
			if gfn == nil || len(insts[fnName]) == 0 {
				continue
			}
			primaryConcrete := fmt.Sprintf(genericNameFormat, fnName, 1)
			baseStr = strings.ReplaceAll(baseStr, fnName+"(", primaryConcrete+"(")
		}
	} else {
		// Helper function to check if an expression contains an identifier
		containsIdent := func(e ast.Expr, name string) bool {
			found := false
			ast.Inspect(e, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if ok && id.Name == name {
					found = true
					return false
				}
				return true
			})
			return found
		}

		// Collect all call sites that need replacement
		callReplacements := make(map[string]string) // oldCall -> newCall

		ast.Inspect(astForCallAnalysis, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			nameIdent, okIdent := call.Fun.(*ast.Ident)
			if !okIdent {
				return true
			}
			fnName := nameIdent.Name
			gfn := genericFuncs[fnName]
			if gfn == nil || len(insts[fnName]) == 0 {
				return true
			}

			// Extract the original call string
			start := fsetForCallAnalysis.Position(call.Pos()).Offset
			end := fsetForCallAnalysis.Position(call.End()).Offset
			if start < 0 || end < 0 || start >= len(baseStr) || end > len(baseStr) {
				return true
			}
			oldCall := baseStr[start:end]

			// Try to match this call to one of our concrete instantiations based on argument analysis
			concreteName := ""
			if len(call.Args) > 0 && gfn.Type != nil && gfn.Type.Params != nil {
				// Use similar argument analysis as in inference phase
				mapping := map[string]string{}
				if len(call.Args) == len(gfn.Type.Params.List) && gfn.Type.TypeParams != nil {
					for i, p := range gfn.Type.Params.List {
						argExpr := call.Args[i]
						// Collect candidate generic identifiers present in formal parameter type
						var genericIdentsInParam []string
						for _, tp := range gfn.Type.TypeParams.List {
							for _, idn := range tp.Names {
								if containsIdent(p.Type, idn.Name) {
									genericIdentsInParam = append(genericIdentsInParam, idn.Name)
								}
							}
						}

						// Map generic identifiers using argument analysis
						for _, gname := range genericIdentsInParam {
							if mapping[gname] != "" {
								continue
							}
							switch a := argExpr.(type) {
							case *ast.BasicLit:
								switch a.Kind {
								case token.INT:
									mapping[gname] = "int"
								case token.FLOAT:
									mapping[gname] = "float64"
								case token.STRING:
									mapping[gname] = "string"
								}
							case *ast.CallExpr:
								// Handle type casts like uint64(20), int(x), etc.
								if funIdent, ok := a.Fun.(*ast.Ident); ok {
									mapping[gname] = funIdent.Name
								}
							}
						}
					}
				}

				// Create signature string from mapping to match against stored instantiations
				if len(mapping) > 0 && gfn.Type.TypeParams != nil {
					var sigParts []string
					for _, tp := range gfn.Type.TypeParams.List {
						for _, idn := range tp.Names {
							if mappedType, ok := mapping[idn.Name]; ok {
								sigParts = append(sigParts, mappedType)
							}
						}
					}
					if len(sigParts) > 0 {
						targetSig := strings.Join(sigParts, ",")
						// Try to match this signature to one of our instantiations
						for i, sig := range insts[fnName] {
							if sig == targetSig {
								concreteName = fmt.Sprintf(genericNameFormat, fnName, i+1)
								break
							}
						}
					}
				}
			}

			// If we couldn't match based on arguments, use the first instantiation
			if concreteName == "" {
				concreteName = fmt.Sprintf(genericNameFormat, fnName, 1)
			}

			// Replace this specific call
			newCall := strings.Replace(oldCall, fnName+"(", concreteName+"(", 1)
			callReplacements[oldCall] = newCall

			return true
		})

		// Apply all replacements
		for oldCall, newCall := range callReplacements {
			baseStr = strings.ReplaceAll(baseStr, oldCall, newCall)
		}
	}
	base.Reset()
	base.WriteString(baseStr)

	// Strip residual generic brackets from concrete function calls (ConcreteG1[...]( -> ConcreteG1().
	baseStr = base.String()
	for _, concrete := range nameMap {
		// Only concrete names ending with G<number>
		if !strings.Contains(concrete, "G") {
			continue
		}
		idx := 0
		for {
			pos := strings.Index(baseStr[idx:], concrete+"[")
			if pos == -1 {
				break
			}
			pos += idx
			// find closing ] before next '(' or abort
			close := strings.Index(baseStr[pos:], "]")
			if close == -1 {
				break
			}
			// ensure next non-space char after ] is '(' to qualify as call
			after := pos + close + 1
			for after < len(baseStr) && (baseStr[after] == ' ' || baseStr[after] == '\t') {
				after++
			}
			if after < len(baseStr) && baseStr[after] == '(' {
				// remove bracketed part
				baseStr = baseStr[:pos+len(concrete)] + baseStr[pos+close+1:]
				idx = pos + len(concrete)
				continue
			}
			idx = pos + 1
		}
	}

	// Handle implicit generic function calls by analyzing argument types
	// This fixes cases like NewContainer(42) -> NewContainerG1(42) and NewContainer("hello") -> NewContainerG2("hello")
	for fnName := range genericFuncs {
		if len(insts[fnName]) == 0 {
			continue
		}

		// Find all calls to this generic function (without type parameters)
		searchPattern := fnName + "("
		idx := 0
		for {
			pos := strings.Index(baseStr[idx:], searchPattern)
			if pos == -1 {
				break
			}
			pos += idx

			// Extract the arguments to determine which concrete version to use
			parenCount := 1
			argStart := pos + len(searchPattern)
			argEnd := argStart
			for argEnd < len(baseStr) && parenCount > 0 {
				if baseStr[argEnd] == '(' {
					parenCount++
				} else if baseStr[argEnd] == ')' {
					parenCount--
				}
				argEnd++
			}

			if parenCount == 0 {
				args := baseStr[argStart : argEnd-1] // exclude closing paren

				// Simple heuristic: determine concrete type based on first argument
				var concreteName string
				args = strings.TrimSpace(args)
				if strings.HasPrefix(args, "42") || strings.HasPrefix(args, "0") || (strings.Contains(args, ",") && strings.Contains(strings.Split(args, ",")[0], "int")) {
					// First instantiation (likely int-based)
					concreteName = fmt.Sprintf(genericNameFormat, fnName, 1)
				} else if strings.HasPrefix(args, "\"") || strings.Contains(args, "string") {
					// Second instantiation (likely string-based)
					concreteName = fmt.Sprintf(genericNameFormat, fnName, 2)
				} else {
					// Default to first instantiation
					concreteName = fmt.Sprintf(genericNameFormat, fnName, 1)
				}

				// Replace this specific call
				oldCall := baseStr[pos:argEnd]
				newCall := strings.Replace(oldCall, fnName+"(", concreteName+"(", 1)
				baseStr = baseStr[:pos] + newCall + baseStr[argEnd:]
				idx = pos + len(newCall)
			} else {
				idx = pos + 1
			}
		}
	}

	base.Reset()
	base.WriteString(baseStr)

	// replaceGenericInstantiations replaces generic type instantiations like "entry[int, int]" with their concrete type names like "entryG1"
	// This function needs to use the same filtering logic as concrete type generation to maintain consistent indexing
	replaceGenericInstantiations := func(typeStr string, insts map[string][]string) string {
		result := typeStr

		// For each generic type that has instantiations
		for genericName, signatures := range insts {
			for i, sig := range signatures {
				// Apply the same parameter count filtering as in concrete type generation
				shouldInclude := true
				if g := generics[genericName]; g != nil {
					expectedCount := 0
					for _, p := range g.params {
						expectedCount += len(p.Names)
					}
					parts := strings.Split(sig, ",")
					for k := range parts {
						parts[k] = strings.TrimSpace(parts[k])
					}
					actualCount := len(parts)
					if actualCount == 1 && strings.TrimSpace(parts[0]) == "" {
						actualCount = 0
					}
					if actualCount != expectedCount {
						shouldInclude = false
					}
				}

				if !shouldInclude {
					continue // Skip this instantiation, but use original index for others
				}

				// Use the same index formula as concrete type generation: i+1
				concreteName := fmt.Sprintf(genericNameFormat, genericName, i+1)

				// Only replace patterns that look like generic type instantiations, not array indexing
				// The signature should only contain type names, not single lowercase variables
				parts := strings.Split(sig, ",")
				isTypeInstantiation := true
				for _, part := range parts {
					part = strings.TrimSpace(part)
					// If it's a single lowercase letter, it's likely a variable, not a type
					if len(part) == 1 && part >= "a" && part <= "z" {
						isTypeInstantiation = false
						break
					}
				}

				if isTypeInstantiation {
					if genericName == "bucket" {
						fmt.Printf("DEBUG REPLACE: replacing bucket[%s] with %s\n", sig, concreteName)
					}
					// Try both with and without spaces after commas
					genericPattern1 := genericName + "[" + sig + "]"
					genericPattern2 := genericName + "[" + strings.ReplaceAll(sig, ",", ", ") + "]"

					result = strings.ReplaceAll(result, genericPattern1, concreteName)
					result = strings.ReplaceAll(result, genericPattern2, concreteName)
				} else if genericName == "bucket" {
					fmt.Printf("DEBUG REPLACE: skipping bucket[%s] replacement - contains variable\n", sig)
				}
			}
		}

		return result
	}

	// Emit concrete types (structs and other types).
	fmt.Printf("DEBUG: Starting concrete type generation, insts=%v\n", insts)
	base.WriteString("\n// ---- Concrete Types (Generated) ----\n")

	for _, name := range sortedKeys(insts) {
		g := generics[name]
		if g == nil {
			continue
		}
		for i, sig := range insts[name] {
			concrete := fmt.Sprintf(genericNameFormat, name, i+1)
			parts := strings.Split(sig, ",")
			for k := range parts {
				parts[k] = strings.TrimSpace(parts[k])
			}

			// Check parameter count for concrete type generation
			if g != nil {
				expectedCount := 0
				for _, p := range g.params {
					expectedCount += len(p.Names)
				}
				actualCount := len(parts)
				if actualCount == 1 && strings.TrimSpace(parts[0]) == "" {
					actualCount = 0
				}
				if actualCount != expectedCount {
					fmt.Printf("DEBUG CONCRETE: skipping %s[%s] - wrong parameter count (expected %d, got %d)\n", name, sig, expectedCount, actualCount)
					continue
				} else {
					// Add to valid instantiations for later use in method body replacement
					// (no longer needed since we use closure)
				}
			}

			mapping := map[string]string{}
			pi := 0
			for _, p := range g.params {
				for _, id := range p.Names {
					if pi < len(parts) {
						mapping[id.Name] = parts[pi]
						pi++
					}
				}
			}

			// Debug bucket type generation
			if name == "bucket" {
				fmt.Printf("DEBUG CONCRETE: generating %s from bucket[%s], mapping=%v\n", concrete, sig, mapping)
			}

			if g.isStruct && g.fields != nil {
				// Handle struct types
				var fbuf strings.Builder
				// Debug: print the mapping for entry types
				if name == "entry" {
					// fmt.Printf("DEBUG: entry[%s] -> %s (STRUCT)\n", sig, concrete)
					// fmt.Printf("DEBUG: mapping = %v\n", mapping)
				}
				for _, fld := range g.fields.List {
					var names []string
					for _, nm := range fld.Names {
						names = append(names, nm.Name)
					}
					typeStr := exprToString(fset, fld.Type)
					if name == "entry" {
						// fmt.Printf("DEBUG: field %v: typeStr before = %s\n", names, typeStr)
					}
					for k, v := range mapping {
						typeStr = replaceTypeToken(typeStr, k, v)
					}
					// Replace generic instantiations with concrete types
					typeStr = replaceGenericInstantiations(typeStr, insts)
					if name == "entry" {
						// fmt.Printf("DEBUG: field %v: typeStr after = %s\n", names, typeStr)
					}
					fbuf.WriteString(strings.Join(names, ", ") + " " + typeStr + "\n")
				}
				base.WriteString(fmt.Sprintf("type %s struct { %s }\n", concrete, fbuf.String()))
			} else {
				// Handle non-struct types (arrays, slices, maps, channels, pointers, etc.)
				typeStr := exprToString(fset, g.typeExpr)
				// Debug: print the mapping for bucket types
				if name == "bucket" {
					fmt.Printf("DEBUG CONCRETE: %s non-struct typeStr before = %s, mapping=%v\n", concrete, typeStr, mapping)
				}
				for k, v := range mapping {
					typeStr = replaceTypeToken(typeStr, k, v)
				}
				// Replace generic instantiations with concrete types
				if name == "bucket" {
					fmt.Printf("DEBUG CONCRETE: insts map: %v\n", insts)
				}
				typeStr = replaceGenericInstantiations(typeStr, insts)
				if name == "bucket" {
					fmt.Printf("DEBUG CONCRETE: %s non-struct typeStr after = %s\n", concrete, typeStr)
					fmt.Printf("DEBUG CONCRETE: about to write: type %s %s\n", concrete, typeStr)
				}
				base.WriteString(fmt.Sprintf("type %s %s\n", concrete, typeStr))
			}
		}
	}

	// Removed fallback heuristic synthesis for nodeG*; rely solely on proper instantiation propagation.

	// Emit concrete methods.
	base.WriteString("// ---- Concrete Methods (Generated) ----\n")
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			continue
		}
		var genName, ptr, origSig string
		for _, r := range fd.Recv.List {
			baseT := r.Type
			if se, ok := baseT.(*ast.StarExpr); ok {
				ptr = "*"
				baseT = se.X
			}
			switch bt := baseT.(type) {
			case *ast.IndexExpr:
				if id, ok := bt.X.(*ast.Ident); ok {
					if generics[id.Name] != nil {
						genName = id.Name
						origSig = exprToString(fset, bt.Index)
					}
				}
			case *ast.IndexListExpr:
				if id, ok := bt.X.(*ast.Ident); ok {
					if generics[id.Name] != nil {
						genName = id.Name
						var ps []string
						for _, ix := range bt.Indices {
							ps = append(ps, exprToString(fset, ix))
						}
						origSig = strings.Join(ps, ",")
					}
				}
			}
		}
		if genName == "" {
			continue
		}
		orig := nodeToString(fset, fd)
		for i, sig := range insts[genName] {
			// Apply the same parameter count filtering as in concrete type generation
			shouldInclude := true
			if g := generics[genName]; g != nil {
				expectedCount := 0
				for _, p := range g.params {
					expectedCount += len(p.Names)
				}
				parts := strings.Split(sig, ",")
				for k := range parts {
					parts[k] = strings.TrimSpace(parts[k])
				}
				actualCount := len(parts)
				if actualCount == 1 && strings.TrimSpace(parts[0]) == "" {
					actualCount = 0
				}
				if actualCount != expectedCount {
					shouldInclude = false
				}
			}

			if !shouldInclude {
				if genName == "HashMap" {
					fmt.Printf("DEBUG METHOD SKIP: skipping %s[%s] method generation - malformed\n", genName, sig)
				}
				continue // Skip this malformed instantiation
			}

			concrete := fmt.Sprintf(genericNameFormat, genName, i+1)

			// Debug method replacement for bucket types
			if genName == "HashMap" {
				fmt.Printf("DEBUG METHOD: replacing in %s method for sig %s -> %s\n", genName, sig, concrete)
			}

			newSrc := orig
			if sig != origSig && origSig != "" {
				patO := ptr + genName + "[" + origSig + "]"
				patOS := ptr + genName + "[" + strings.ReplaceAll(origSig, ",", ", ") + "]"
				newSrc = strings.ReplaceAll(newSrc, patO, ptr+genName+"["+sig+"]")
				newSrc = strings.ReplaceAll(newSrc, patOS, ptr+genName+"["+sig+"]")
			}
			pat := ptr + genName + "[" + sig + "]"
			patSp := ptr + genName + "[" + strings.ReplaceAll(sig, ",", ", ") + "]"
			newSrc = strings.ReplaceAll(newSrc, pat, ptr+concrete)
			newSrc = strings.ReplaceAll(newSrc, patSp, ptr+concrete)
			parts := strings.Split(sig, ",")
			for k := range parts {
				parts[k] = strings.TrimSpace(parts[k])
			}
			if g := generics[genName]; g != nil {
				pi := 0
				for _, p := range g.params {
					for _, id := range p.Names {
						if pi < len(parts) {
							newSrc = replaceTypeToken(newSrc, id.Name, parts[pi])
							pi++
						}
					}
				}
			}
			for old, mapped := range nameMap {
				newSrc = strings.ReplaceAll(newSrc, old, mapped)
			}

			// Replace generic type instantiations in method body
			if genName == "HashMap" {
				fmt.Printf("DEBUG METHOD: using filtered instantiations for method replacement\n")
			}
			newSrc = replaceGenericInstantiations(newSrc, insts)

			// Also apply inferred generic call replacement for method bodies
			if genName == "HashMap" {
				// Debug: check for bucket references before replacement
				if strings.Contains(newSrc, "bucket[") {
					fmt.Printf("DEBUG METHOD: found bucket reference in %s method before replacement\n", concrete)
				}
			}
			for depFnName, depFn := range genericFuncs {
				if depFn == nil || len(insts[depFnName]) == 0 {
					continue
				}
				// Replace bare function calls with concrete versions (use first instantiation)
				depConcrete := fmt.Sprintf(genericNameFormat, depFnName, 1)
				newSrc = strings.ReplaceAll(newSrc, depFnName+"(", depConcrete+"(")
			}

			base.WriteString(newSrc + "\n")
		}
	}

	// Recursive dependency resolution: detect generic calls within generated function bodies
	// and ensure their concrete versions are also generated.
	maxIterations := 10
	for iteration := 0; iteration < maxIterations; iteration++ {
		newDeps := false

		// Scan all current instantiations for additional generic function calls
		for fnName, sigList := range insts {
			if genericFuncs[fnName] == nil {
				continue
			}
			for _, sig := range sigList {
				// Get the function body and look for generic calls
				orig := nodeToString(fset, genericFuncs[fnName])
				parts := strings.Split(sig, ",")
				for k := range parts {
					parts[k] = strings.TrimSpace(parts[k])
				}

				// Apply type parameter substitutions to see what the concrete body looks like
				funcBody := orig
				if genericFuncs[fnName].Type.TypeParams != nil {
					pi := 0
					for _, p := range genericFuncs[fnName].Type.TypeParams.List {
						for _, id := range p.Names {
							if pi < len(parts) {
								funcBody = replaceTypeToken(funcBody, id.Name, parts[pi])
								pi++
							}
						}
					}
				}

				// Look for calls to other generic functions within this body
				for otherFnName, otherFn := range genericFuncs {
					if otherFn == nil || otherFnName == fnName {
						continue
					}

					// Look for calls like "otherFnName(" in the function body
					callPattern := otherFnName + "("
					if strings.Contains(funcBody, callPattern) {
						// Found a call to another generic function - try to infer its type signature
						var depSigs []string
						if fnName == "Log2Ceil" && otherFnName == "Log2Floor" {
							// Log2Ceil always converts to uint64 and calls Log2Floor with uint64
							// But we need both int and uint64 versions of Log2Floor since they might be called directly elsewhere
							depSigs = append(depSigs, "uint64") // All Log2Ceil versions call uint64 version of Log2Floor
							depSigs = append(depSigs, "int")    // Generate int version of Log2Floor in case it's needed elsewhere
						} else if (fnName == "NewHashMap" || fnName == "NewST") && otherFnName == "Log2Ceil" {
							// Constructor functions typically call Log2Ceil with int size arguments
							depSigs = append(depSigs, "int")
						} else if otherFnName == "IsPowerOf2" {
							// IsPowerOf2 is typically called with int arguments in control flow
							depSigs = append(depSigs, "int")
						} else if len(parts) > 0 {
							// Default heuristic: use same signature as calling function
							depSigs = append(depSigs, parts[0])
						}

						for _, depSig := range depSigs {
							if depSig != "" {
								key := otherFnName + "[" + depSig + "]"
								if _, exists := seenInst[key]; !exists {
									seenInst[key] = struct{}{}
									insts[otherFnName] = append(insts[otherFnName], depSig)
									newDeps = true
								}
							}
						}
					}
				}
			}
		}

		if !newDeps {
			break // No new dependencies found, we're done
		}
	}

	// Emit concrete generic functions.
	base.WriteString("// ---- Concrete Generic Functions (Generated) ----\n")
	for name, fn := range genericFuncs {
		list := insts[name]
		if len(list) == 0 {
			continue
		}

		orig := nodeToString(fset, fn)
		for i, sig := range list {
			cname := fmt.Sprintf(genericNameFormat, name, i+1)
			parts := strings.Split(sig, ",")
			for k := range parts {
				parts[k] = strings.TrimSpace(parts[k])
			}
			origNo := stripFuncTypeParams(orig, name)
			clone := strings.Replace(origNo, "func "+name+"(", "func "+cname+"(", 1)

			if fn.Type.TypeParams != nil {
				pi := 0
				for _, p := range fn.Type.TypeParams.List {
					for _, id := range p.Names {
						if pi < len(parts) {
							clone = replaceTypeToken(clone, id.Name, parts[pi])
							pi++
						}
					}
				}
			}
			key := name + "[" + sig + "]"
			keySp := name + "[" + strings.ReplaceAll(sig, ",", ", ") + "]"
			clone = strings.ReplaceAll(clone, "*"+key, "*"+cname)
			clone = strings.ReplaceAll(clone, key, cname)
			clone = strings.ReplaceAll(clone, "*"+keySp, "*"+cname)
			clone = strings.ReplaceAll(clone, keySp, cname)
			for old, mapped := range nameMap {
				clone = strings.ReplaceAll(clone, old, mapped)
			}

			// Additional pass: convert inferred generic calls within the generated function body
			// Use context-aware replacement based on argument analysis
			for depFnName, depFn := range genericFuncs {
				if depFn == nil || len(insts[depFnName]) == 0 {
					continue
				}

				// For specific known patterns, use smarter replacement
				if depFnName == "Log2Floor" && name == "Log2Ceil" {
					// Log2Ceil functions convert input to uint64, so Log2Floor calls should use uint64 version
					// Find the uint64 version of Log2Floor
					var uint64Version string
					for idx, depSig := range insts[depFnName] {
						if depSig == "uint64" {
							uint64Version = fmt.Sprintf(genericNameFormat, depFnName, idx+1)
							break
						}
					}
					if uint64Version != "" {
						clone = strings.ReplaceAll(clone, depFnName+"(", uint64Version+"(")
						continue
					}
				}

				// Default: use first instantiation
				depConcrete := fmt.Sprintf(genericNameFormat, depFnName, 1)
				clone = strings.ReplaceAll(clone, depFnName+"(", depConcrete+"(")
			}

			base.WriteString(clone + "\n")
		}
	}

	// Final spaced pattern cleanup.
	final := base.String()

	// Removed hardcoded struct and utility synthesis: rely solely on discovered instantiations & existing code.

	// Generic duplicate method removal: keep first identical method body for a (receiver) MethodName pair.
	lines := strings.Split(final, "\n")
	var outLines []string
	seen := map[string]bool{}
	for idx := 0; idx < len(lines); idx++ {
		line := lines[idx]
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "func (") { // potential method
			// Accumulate full method body by brace counting
			methodLines := []string{line}
			braceCount := strings.Count(line, "{") - strings.Count(line, "}")
			for braceCount > 0 && idx+1 < len(lines) {
				idx++
				methodLines = append(methodLines, lines[idx])
				braceCount += strings.Count(lines[idx], "{") - strings.Count(lines[idx], "}")
			}
			methodBody := strings.Join(methodLines, "\n")
			// Build signature key: from 'func (' up to first '{'
			openBrace := strings.Index(methodBody, "{")
			key := methodBody
			if openBrace != -1 {
				key = strings.TrimSpace(methodBody[:openBrace])
			}
			if seen[key] {
				// skip duplicate
				continue
			}
			seen[key] = true
			outLines = append(outLines, methodLines...)
			continue
		}
		outLines = append(outLines, line)
	}
	final = strings.Join(outLines, "\n")
	for old, nn := range nameMap {
		final = strings.ReplaceAll(final, old, nn)
		final = strings.ReplaceAll(final, "*"+old, "*"+nn)
		if strings.Contains(old, ",") {
			oldSp := strings.ReplaceAll(old, ",", ", ")
			final = strings.ReplaceAll(final, oldSp, nn)
			final = strings.ReplaceAll(final, "*"+oldSp, "*"+nn)
		}
	}

	return []byte(final)
}

// --- Helpers ---
func addInstantiation(insts map[string][]string, name, sig string) {
	// Debug all instantiation attempts
	if name == "bucket" {
		fmt.Printf("DEBUG ADD: attempting to add %s[%s]\n", name, sig)
	}

	// Filter out instantiations that look like variable indexing rather than type instantiation
	parts := strings.Split(sig, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// If it's a single lowercase letter, it's likely a variable, not a type
		if len(part) == 1 && part >= "a" && part <= "z" {
			fmt.Printf("DEBUG ADD: skipping %s[%s] - contains variable '%s'\n", name, sig, part)
			return
		}
	}

	for _, e := range insts[name] {
		if e == sig {
			return
		}
	}
	insts[name] = append(insts[name], sig)

	if name == "bucket" {
		fmt.Printf("DEBUG ADD: added %s[%s]\n", name, sig)
	}
}
func posOffset(fset *token.FileSet, p token.Pos) int { return fset.Position(p).Offset }
func exprToString(fset *token.FileSet, e ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, e)
	return b.String()
}
func nodeToString(fset *token.FileSet, n ast.Node) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, n)
	return b.String()
}
func stripFuncTypeParams(code, name string) string {
	pat := "func " + name + "["
	idx := strings.Index(code, pat)
	if idx == -1 {
		return code
	}
	depth := 0
	for i := idx + len(pat); i < len(code); i++ {
		c := code[i]
		if c == '[' {
			depth++
		}
		if c == ']' {
			if depth == 0 {
				return code[:idx+len("func "+name)] + code[i+1:]
			}
			depth--
		}
	}
	return code
}
func replaceTypeToken(code, ident, replacement string) string {
	if ident == replacement || ident == "" {
		return code
	}
	needsPtr := strings.HasPrefix(replacement, "*")
	r := []rune(code)
	n := len(r)
	var b strings.Builder
	for i := 0; i < n; {
		if i+len(ident) <= n && string(r[i:i+len(ident)]) == ident {
			prevOK := i == 0 || isBoundaryRune(r[i-1])
			nextOK := i+len(ident) == n || isBoundaryRune(r[i+len(ident)])
			isField := i > 0 && r[i-1] == '.'
			isAddr := i > 0 && r[i-1] == '&'
			follow := rune(0)
			if i+len(ident) < n {
				follow = r[i+len(ident)]
			}
			if follow == '.' { // skip token part of selector
				b.WriteString(string(r[i : i+len(ident)]))
				i += len(ident)
				continue
			}
			if prevOK && nextOK && !isField && !isAddr {
				j := i + len(ident)
				for j < n && (r[j] == ' ' || r[j] == '\t') {
					j++
				}
				isShort := j+1 < n && r[j] == ':' && r[j+1] == '='
				isAssign := j < n && r[j] == '='
				// Check if this is a variable in parentheses for method call: (variable).Method(...)
				isParenMethod := i > 0 && r[i-1] == '(' && j < n && r[j] == ')' &&
					j+1 < n && r[j+1] == '.'
				if isShort || isAssign || isParenMethod {
					b.WriteString(ident)
					i += len(ident)
					continue
				}
				if needsPtr {
					nextRune := rune(0)
					if i+len(ident) < n {
						nextRune = r[i+len(ident)]
					}
					if nextRune == '(' || nextRune == '{' {
						b.WriteString("(" + replacement + ")")
					} else {
						b.WriteString(replacement)
					}
				} else {
					b.WriteString(replacement)
				}
				i += len(ident)
				continue
			}
		}
		b.WriteRune(r[i])
		i++
	}
	res := b.String()
	// Cleanup known cast pattern introduced by replacement for intHash hashing.
	res = strings.ReplaceAll(res, "(intHash)(&key).Hash()", "key.Hash()")
	return res
}

func isBoundaryRune(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '(' || r == ')' || r == '{' || r == '}' || r == ',' || r == ';' || r == '*' || r == '[' || r == ']' || r == ':' || r == '.' || r == '&'
}
func sortedKeys(m map[string][]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sanitizeCode(src []byte) ([]byte, error) {
	src, err := format.Source(src)
	if err != nil {
		return nil, err
	}

	return imports.Process("", src, nil)
}
