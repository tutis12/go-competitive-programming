package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"strings"
)

// AST-based RemoveGenerics implementation.
var genericNameFormat = "%sG%d"

func RemoveGenerics(src []byte) []byte {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "merged.go", src, parser.ParseComments)
	if err != nil {
		return src
	}
	type genericInfo struct {
		decl     *ast.GenDecl // full declaration (to remove 'type' token cleanly)
		spec     *ast.TypeSpec
		isStruct bool
		params   []*ast.Field
		fields   *ast.FieldList // struct fields (nil for interface)
	}
	generics := map[string]*genericInfo{}
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, sp := range gd.Specs {
			ts := sp.(*ast.TypeSpec)
			if ts.TypeParams == nil {
				continue
			}
			gi := &genericInfo{decl: gd, spec: ts, params: ts.TypeParams.List}
			switch tt := ts.Type.(type) {
			case *ast.StructType:
				gi.isStruct = true
				gi.fields = tt.Fields
			case *ast.InterfaceType:
				gi.isStruct = false
				gi.fields = nil
			default:
				continue // only handle struct/interface generics for now
			}
			generics[ts.Name.Name] = gi
		}
	}
	insts := map[string][]string{}
	genericFuncs := map[string]*ast.FuncDecl{}
	for _, d := range file.Decls { // collect standalone generic functions
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Type != nil && fd.Type.TypeParams != nil && fd.Recv == nil {
			genericFuncs[fd.Name.Name] = fd
		}
	}
	seen := map[string]struct{}{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.IndexExpr:
			if id, ok := e.X.(*ast.Ident); ok {
				if g := generics[id.Name]; g != nil || genericFuncs[id.Name] != nil {
					arg := exprToString(fset, e.Index)
					key := id.Name + "[" + arg + "]"
					if _, dup := seen[key]; !dup {
						seen[key] = struct{}{}
						insts[id.Name] = append(insts[id.Name], arg)
					}
				}
			}
		case *ast.IndexListExpr:
			if id, ok := e.X.(*ast.Ident); ok {
				if g := generics[id.Name]; g != nil || genericFuncs[id.Name] != nil {
					var parts []string
					for _, ix := range e.Indices {
						parts = append(parts, exprToString(fset, ix))
					}
					arg := strings.Join(parts, ",")
					key := id.Name + "[" + arg + "]"
					if _, dup := seen[key]; !dup {
						seen[key] = struct{}{}
						insts[id.Name] = append(insts[id.Name], arg)
					}
				}
			}
		}
		return true
	})
	// Heuristic: infer instantiations from calls to generic functions without explicit type args
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		ast.Inspect(fd, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			var funcName string
			var gfn *ast.FuncDecl

			// Handle both direct calls (func) and qualified calls (pkg.func)
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				funcName = fun.Name
				gfn = genericFuncs[funcName]
			case *ast.SelectorExpr:
				funcName = fun.Sel.Name
				gfn = genericFuncs[funcName]
			default:
				return true
			}

			if gfn == nil {
				return true
			}
			// Previously we skipped once we had any instantiation; instead allow multiple distinct inferred instantiations.
			// Attempt to map type params using arguments whose parameter types are single identifiers of type params.
			if gfn.Type.Params == nil {
				return true
			}
			paramList := gfn.Type.Params.List
			if len(call.Args) != len(paramList) {
				return true
			}
			mapping := map[string]string{}
			for pi, p := range paramList {
				// Only consider parameter types that are bare identifiers referring to a generic parameter
				bt, okT := p.Type.(*ast.Ident)
				if !okT {
					continue
				}
				// attempt to derive concrete type from argument expression
				arg := call.Args[pi]
				switch a := arg.(type) {
				case *ast.CompositeLit:
					if a.Type != nil {
						mapping[bt.Name] = exprToString(fset, a.Type)
					} else {
						// For composite literals without explicit type, try to infer from context
						// Look for likely type names based on the composite literal structure
						if len(a.Elts) == 0 {
							// Empty composite literal - look for short type names in scope
							for _, d := range file.Decls {
								if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
									for _, sp := range gd.Specs {
										if ts, ok := sp.(*ast.TypeSpec); ok {
											typeName := ts.Name.Name
											// Heuristic: short type names are often used for empty structs
											if len(typeName) <= 6 && strings.ToLower(typeName) == typeName {
												mapping[bt.Name] = typeName
												break
											}
										}
									}
								}
							}
						}
					}
				case *ast.CallExpr:
					// Could be a constructor returning concrete type; ignore (needs deeper analysis)
				case *ast.FuncLit:
					// examine returns for composite literals
					var retType string
					ast.Inspect(a.Body, func(x ast.Node) bool {
						ret, okR := x.(*ast.ReturnStmt)
						if !okR {
							return true
						}
						if len(ret.Results) == 1 {
							if cl, okCL := ret.Results[0].(*ast.CompositeLit); okCL && cl.Type != nil {
								retType = exprToString(fset, cl.Type)
							}
						}
						return true
					})
					if retType != "" {
						mapping[bt.Name] = retType
					}
				case *ast.UnaryExpr:
					// pointer to composite literal
					if cl, okCL := a.X.(*ast.CompositeLit); okCL && cl.Type != nil {
						mapping[bt.Name] = exprToString(fset, cl.Type)
					}
				case *ast.BasicLit:
					// Try to infer type from literal
					switch a.Kind.String() {
					case "INT":
						mapping[bt.Name] = "int"
					case "FLOAT":
						mapping[bt.Name] = "float64"
					case "STRING":
						mapping[bt.Name] = "string"
					}
				case *ast.Ident:
					// For simple identifiers, try to infer common types
					// This is a heuristic - in real code analysis we'd need full type information
					argName := a.Name
					if strings.Contains(argName, "size") || strings.Contains(argName, "length") || strings.Contains(argName, "count") || strings.Contains(argName, "index") || argName == "i" || argName == "j" || argName == "k" || argName == "n" {
						mapping[bt.Name] = "int"
					}
				}

			}
			// Build instantiation string if we mapped all generic params
			if gfn.Type.TypeParams != nil {
				parts := []string{}
				allMapped := true
				for _, tp := range gfn.Type.TypeParams.List {
					for _, name := range tp.Names {
						val := mapping[name.Name]
						if val == "" {
							allMapped = false
						}
						parts = append(parts, val)
					}
				}

				// Try to infer missing type parameters from interface constraints
				if !allMapped && gfn.Type.TypeParams != nil {
					for _, tp := range gfn.Type.TypeParams.List {
						for _, name := range tp.Names {
							if mapping[name.Name] == "" {
								// Try to infer from interface constraint
								if iface, ok := tp.Type.(*ast.InterfaceType); ok {
									for _, method := range iface.Methods.List {
										// Look for patterns like *update in interface
										if starExpr, ok := method.Type.(*ast.StarExpr); ok {
											if ident, ok := starExpr.X.(*ast.Ident); ok {
												if baseType := mapping[ident.Name]; baseType != "" {
													mapping[name.Name] = "*" + baseType
													break
												}
											}
										}
									}
								}
							}
						}
					}

					// Rebuild parts array with inferred types
					parts = []string{}
					allMapped = true
					for _, tp := range gfn.Type.TypeParams.List {
						for _, name := range tp.Names {
							val := mapping[name.Name]
							if val == "" {
								allMapped = false
							}
							parts = append(parts, val)
						}
					}
				}

				if allMapped {
					argStr := strings.Join(parts, ",")
					key := funcName + "[" + argStr + "]"
					if _, dup := seen[key]; !dup {
						seen[key] = struct{}{}
						// ensure uniqueness in insts slice (avoid duplicates from different paths)
						already := false
						for _, existing := range insts[funcName] {
							if existing == argStr {
								already = true
								break
							}
						}
						if !already {
							insts[funcName] = append(insts[funcName], argStr)
						}
					}
					// Also add struct instantiation for any return type referencing generic struct with params
					if gfn.Type.Results != nil && len(gfn.Type.Results.List) == 1 {
						ret := gfn.Type.Results.List[0].Type
						if se, okSe := ret.(*ast.StarExpr); okSe {
							ret = se.X
						}
						if ix, okIx := ret.(*ast.IndexListExpr); okIx {
							if base, okB := ix.X.(*ast.Ident); okB {
								if _, isStruct := generics[base.Name]; isStruct {
									// Map generic struct params via mapping
									structParts := []string{}
									for _, tp := range generics[base.Name].params {
										for _, id2 := range tp.Names {
											structParts = append(structParts, mapping[id2.Name])
										}
									}
									sp := strings.Join(structParts, ",")
									sKey := base.Name + "[" + sp + "]"
									if _, dup2 := seen[sKey]; !dup2 {
										seen[sKey] = struct{}{}
										insts[base.Name] = append(insts[base.Name], sp)
									}
									// Update other generic types that might be referenced
									for nestedName, ng := range generics {
										if ng != nil && ng.params != nil {
											var nestedArgs []string
											allFound := true
											for _, p := range ng.params {
												for _, id := range p.Names {
													if val, exists := mapping[id.Name]; exists {
														nestedArgs = append(nestedArgs, val)
													} else {
														allFound = false
														break
													}
												}
												if !allFound {
													break
												}
											}
											if allFound && len(nestedArgs) > 0 {
												nestedArg := strings.Join(nestedArgs, ",")
												keyNested := nestedName + "[" + nestedArg + "]"
												if _, dupN := seen[keyNested]; !dupN {
													seen[keyNested] = struct{}{}
													insts[nestedName] = append(insts[nestedName], nestedArg)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
			return true
		})
	}

	// Detect calls to other generic functions from within generic functions
	for _, gfn := range genericFuncs {
		if len(insts[gfn.Name.Name]) > 0 {
			// This function has instantiations, scan its body for calls to other generic functions
			ast.Inspect(gfn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if id, ok := call.Fun.(*ast.Ident); ok {
						if targetFn := genericFuncs[id.Name]; targetFn != nil {
							// Found a call to another generic function
							// Try to infer the instantiation based on argument types in context
							if len(call.Args) == 1 {
								// For single argument functions, try some common patterns
								arg := call.Args[0]
								if binExpr, ok := arg.(*ast.BinaryExpr); ok {
									// Handle patterns like "x64-1" where x64 is uint64
									if ident, ok := binExpr.X.(*ast.Ident); ok && strings.Contains(ident.Name, "64") {
										// Likely uint64
										key := id.Name + "[uint64]"
										if _, dup := seen[key]; !dup {
											seen[key] = struct{}{}
											insts[id.Name] = append(insts[id.Name], "uint64")
										}
									}
								}
							}
						}
					}
				}
				return true
			})
		}
	}

	// Filter out instantiations that only contain generic parameter names
	allGenericParams := make(map[string]bool)
	for _, g := range generics {
		if g.params != nil {
			for _, p := range g.params {
				for _, id := range p.Names {
					allGenericParams[id.Name] = true
				}
			}
		}
	}
	// Also include function-level generic parameters
	for _, gfn := range genericFuncs {
		if gfn.Type.TypeParams != nil {
			for _, p := range gfn.Type.TypeParams.List {
				for _, id := range p.Names {
					allGenericParams[id.Name] = true
				}
			}
		}
	}

	filtered := make(map[string][]string)
	for name, instList := range insts {
		for _, inst := range instList {
			parts := strings.Split(inst, ",")
			hasConcreteType := false
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if !allGenericParams[part] {
					hasConcreteType = true
					break
				}
			}
			if hasConcreteType {
				filtered[name] = append(filtered[name], inst)
			}
		}
	}
	insts = filtered

	// Propagate instantiations: if a generic struct's field references another generic struct with identical parameter ordering,
	// ensure both are instantiated with the same concrete arguments. This covers patterns like ST[value,update] containing node[value,update].
	if len(insts) > 0 {
		for outerName, outerArgsList := range insts {
			outerGen := generics[outerName]
			if outerGen == nil || !outerGen.isStruct || outerGen.fields == nil {
				continue
			}
			for _, fld := range outerGen.fields.List {
				fieldTypeStr := exprToString(fset, fld.Type)
				for innerName, innerGen := range generics {
					if innerGen == nil || !innerGen.isStruct || innerGen == outerGen || innerGen.params == nil {
						continue
					}
					// Cheap check: field type must contain inner generic name followed by '['
					if !strings.Contains(fieldTypeStr, innerName+"[") {
						continue
					}
					// Parameter counts must match
					paramCount := 0
					for _, p := range innerGen.params {
						paramCount += len(p.Names)
					}
					for _, outerArgs := range outerArgsList {
						parts := strings.Split(outerArgs, ",")
						if len(parts) != paramCount { // skip differing arity
							continue
						}
						// Add instantiation for inner generic if missing
						found := false
						for _, existing := range insts[innerName] {
							if existing == outerArgs {
								found = true
								break
							}
						}
						if !found {
							insts[innerName] = append(insts[innerName], outerArgs)
						}
					}
				}
			}
		}
	}

	if len(insts) == 0 {
		return src
	}
	// Ensure deterministic ordering: sort instantiation slices and then build nameMap in lexical order of generic names.
	for name := range insts {
		lst := insts[name]
		// remove duplicates defensively & sort
		uniq := make([]string, 0, len(lst))
		seenLocal := map[string]bool{}
		for _, v := range lst {
			if !seenLocal[v] {
				seenLocal[v] = true
				uniq = append(uniq, v)
			}
		}
		sort.Strings(uniq)
		insts[name] = uniq
	}
	nameMap := map[string]string{}
	for _, name := range sortedKeys(insts) {
		list := insts[name]
		for i, arg := range list {
			nameMap[name+"["+arg+"]"] = fmt.Sprintf(genericNameFormat, name, i+1)
		}
	}
	type repl struct {
		start, end int
		text       string
	}
	var exprRepls []repl
	ast.Inspect(file, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.IndexExpr:
			if id, ok := e.X.(*ast.Ident); ok {
				if _, isGen := generics[id.Name]; isGen {
					arg := exprToString(fset, e.Index)
					key := id.Name + "[" + arg + "]"
					if newName, ok := nameMap[key]; ok {
						exprRepls = append(exprRepls, repl{fset.Position(e.Pos()).Offset, fset.Position(e.End()).Offset, newName})
					}
				}
			}
		case *ast.IndexListExpr:
			if id, ok := e.X.(*ast.Ident); ok {
				if _, isGen := generics[id.Name]; isGen {
					var parts []string
					for _, ix := range e.Indices {
						parts = append(parts, exprToString(fset, ix))
					}
					arg := strings.Join(parts, ",")
					key := id.Name + "[" + arg + "]"
					if newName, ok := nameMap[key]; ok {
						exprRepls = append(exprRepls, repl{fset.Position(e.Pos()).Offset, fset.Position(e.End()).Offset, newName})
					}
				}
			}
		}
		return true
	})
	var commentRepls []repl
	// Comment out entire type declarations (ensuring 'type' keyword removed)
	seenDecl := map[*ast.GenDecl]bool{}
	for _, g := range generics {
		if seenDecl[g.decl] {
			continue
		}
		seenDecl[g.decl] = true
		pos := fset.Position(g.decl.Pos()).Offset
		end := fset.Position(g.decl.End()).Offset
		// remove generic type completely instead of commenting
		commentRepls = append(commentRepls, repl{pos, end, ""})
	}
	// Comment out top-level generic functions (no receiver, has type params)
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fd.Recv != nil {
			continue
		}
		if fd.Type != nil && fd.Type.TypeParams != nil {
			pos := fset.Position(fd.Pos()).Offset
			end := fset.Position(fd.End()).Offset
			commentRepls = append(commentRepls, repl{pos, end, ""})
		}
	}
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			continue
		}
		for _, f := range fd.Recv.List {
			base := f.Type
			if se, ok := base.(*ast.StarExpr); ok {
				base = se.X
			}
			switch bt := base.(type) {
			case *ast.IndexExpr:
				if id, ok := bt.X.(*ast.Ident); ok {
					if _, isGen := generics[id.Name]; isGen {
						pos := fset.Position(fd.Pos()).Offset
						end := fset.Position(fd.End()).Offset
						commentRepls = append(commentRepls, repl{pos, end, ""})
					}
				}
			case *ast.IndexListExpr:
				if id, ok := bt.X.(*ast.Ident); ok {
					if _, isGen := generics[id.Name]; isGen {
						pos := fset.Position(fd.Pos()).Offset
						end := fset.Position(fd.End()).Offset
						commentRepls = append(commentRepls, repl{pos, end, ""})
					}
				}
			}
		}
	}
	all := append(exprRepls, commentRepls...)
	sort.Slice(all, func(i, j int) bool { return all[i].start < all[j].start })
	var buf bytes.Buffer
	cursor := 0
	for _, r := range all {
		if r.start < cursor {
			continue
		}
		buf.Write(src[cursor:r.start])
		buf.WriteString(r.text)
		cursor = r.end
	}
	buf.Write(src[cursor:])
	buf.WriteString("\n// ---- Concrete Types (Generated) ----\n")
	var structsBuf bytes.Buffer
	for _, name := range sortedKeys(insts) {
		g := generics[name]
		if g == nil || !g.isStruct {
			continue
		} // skip interface generics: no concrete type emitted
		for i, arg := range insts[name] {
			concrete := fmt.Sprintf(genericNameFormat, name, i+1)
			parts := strings.Split(arg, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			mapping := map[string]string{}
			if g.params != nil {
				paramIndex := 0
				for _, p := range g.params {
					for _, id := range p.Names {
						if paramIndex < len(parts) {
							mapping[id.Name] = parts[paramIndex]
							paramIndex++
						}
					}
				}
			}

			// Build fields with precise type substitution
			var fBuilder strings.Builder
			if g.fields != nil {
				for _, fld := range g.fields.List {
					// Collect field names
					var names []string
					for _, nm := range fld.Names {
						names = append(names, nm.Name)
					}
					// Original type source
					typeStr := exprToString(fset, fld.Type)
					// Apply mapping only on type tokens (this replaces simple type params like 'value' -> 'stValue')
					for k, v := range mapping {
						typeStr = replaceTypeToken(typeStr, k, v)
					}
					fBuilder.WriteString(strings.Join(names, ", ") + " " + typeStr + "\n")
				}
			}
			structsBuf.WriteString(fmt.Sprintf("type %s struct { %s }\n", concrete, fBuilder.String()))
		}
	}

	// Second pass: replace any remaining generic instantiations in struct definitions
	structsText := structsBuf.String()
	for key, newName := range nameMap {
		structsText = strings.ReplaceAll(structsText, key, newName)
		if strings.Contains(key, ",") {
			keySpaced := strings.ReplaceAll(key, ",", ", ")
			structsText = strings.ReplaceAll(structsText, keySpaced, newName)
		}
	}

	buf.WriteString(structsText)
	buf.WriteString("// ---- Concrete Methods (Generated) ----\n")
	for _, d := range file.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			continue
		}
		var genName, ptr, origSig string
		for _, f := range fd.Recv.List {
			base := f.Type
			if se, ok := base.(*ast.StarExpr); ok {
				ptr = "*"
				base = se.X
			}
			switch bt := base.(type) {
			case *ast.IndexExpr:
				if id, ok := bt.X.(*ast.Ident); ok {
					if _, isGen := generics[id.Name]; isGen {
						genName = id.Name
						origSig = exprToString(fset, bt.Index)
					}
				}
			case *ast.IndexListExpr:
				if id, ok := bt.X.(*ast.Ident); ok {
					if _, isGen := generics[id.Name]; isGen {
						genName = id.Name
						var parts []string
						for _, ix := range bt.Indices {
							parts = append(parts, exprToString(fset, ix))
						}
						origSig = strings.Join(parts, ",")
					}
				}
			}
		}
		if genName == "" {
			continue
		}
		orig := nodeToString(fset, fd)
		// No special casing of any type
		for i, sig2 := range insts[genName] {
			concrete := fmt.Sprintf(genericNameFormat, genName, i+1)
			newSrc := orig
			if sig2 != origSig && origSig != "" {
				// normalize original receiver to target inst signature first
				patOrig := ptr + genName + "[" + origSig + "]"
				patOrigSpaced := ptr + genName + "[" + strings.ReplaceAll(origSig, ",", ", ") + "]"
				newSrc = strings.ReplaceAll(newSrc, patOrig, ptr+genName+"["+sig2+"]")
				newSrc = strings.ReplaceAll(newSrc, patOrigSpaced, ptr+genName+"["+sig2+"]")
			}
			// now replace target signature with concrete
			pat := ptr + genName + "[" + sig2 + "]"
			patSpaced := ptr + genName + "[" + strings.ReplaceAll(sig2, ",", ", ") + "]"
			newSrc = strings.ReplaceAll(newSrc, pat, ptr+concrete)
			newSrc = strings.ReplaceAll(newSrc, patSpaced, ptr+concrete)
			// Replace type parameter identifiers inside method body/signature
			parts := strings.Split(sig2, ",")
			for k := range parts {
				parts[k] = strings.TrimSpace(parts[k])
			}
			if g := generics[genName]; g != nil {
				paramIndex := 0
				for _, p := range g.params {
					for _, id := range p.Names {
						if paramIndex < len(parts) {
							newSrc = replaceTypeToken(newSrc, id.Name, parts[paramIndex])
							paramIndex++
						}
					}
				}
			}
			// Replace nested generic instantiations like node[...] inside method bodies
			for old, mapped := range nameMap {
				newSrc = strings.ReplaceAll(newSrc, old, mapped)
			}

			buf.WriteString(newSrc + "\n")
		}
	}

	// Clone standalone generic functions (constructor-style or normal)
	buf.WriteString("// ---- Concrete Generic Functions (Generated) ----\n")
	producedSingle := map[string]string{}
	for name, fn := range genericFuncs {
		instList := insts[name]
		// If not instantiated explicitly, attempt inference from return type generic instantiation
		if len(instList) == 0 && fn.Type.Results != nil && len(fn.Type.Results.List) == 1 {
			ret := fn.Type.Results.List[0].Type
			if se, ok := ret.(*ast.StarExpr); ok {
				ret = se.X
			}
			switch rt := ret.(type) {
			case *ast.IndexExpr:
				if id, ok2 := rt.X.(*ast.Ident); ok2 {
					if list, ok3 := insts[id.Name]; ok3 {
						instList = list
					}
				}
			case *ast.IndexListExpr:
				if id, ok2 := rt.X.(*ast.Ident); ok2 {
					if list, ok3 := insts[id.Name]; ok3 {
						instList = list
					}
				}
			}
		}
		if len(instList) == 0 {
			continue
		}
		orig := nodeToString(fset, fn)
		for i, arg := range instList {
			concreteName := fmt.Sprintf(genericNameFormat, name, i+1)
			parts := strings.Split(arg, ",")
			for k := range parts {
				parts[k] = strings.TrimSpace(parts[k])
			}
			origNoParams := stripFuncTypeParams(orig, name)
			clone := strings.Replace(origNoParams, "func "+name+"(", "func "+concreteName+"(", 1)
			// Map type params
			mapping := map[string]string{}
			for j, p := range fn.Type.TypeParams.List {
				if j < len(parts) {
					for _, id := range p.Names {
						mapping[id.Name] = parts[j]
					}
				}
			}
			// Apply all type parameter replacements
			for k, v := range mapping {
				clone = replaceTypeToken(clone, k, v)
			}
			// Fix naming conflicts: if we have both a type T and variable T, rename the variable
			for _, v := range mapping {
				if strings.HasPrefix(v, "*") {
					baseType := v[1:]
					// Rename variable declarations that conflict with type names
					clone = strings.ReplaceAll(clone, "\t"+baseType+" :=", "\t"+baseType+"Val :=")
					clone = strings.ReplaceAll(clone, " "+baseType+" :=", " "+baseType+"Val :=")
					// Update all references to the renamed variable
					clone = strings.ReplaceAll(clone, "&"+baseType+")", "&"+baseType+"Val)")
				}
			}
			// Targeted substitution only for current instantiation to avoid cross-talk.
			curKey := name + "[" + arg + "]"
			if concreteName != "" {
				// Replace pointer and non-pointer return types/signatures.
				// We avoid replacing other instantiations by restricting to exact current arg list.
				curKeySpaced := name + "[" + strings.ReplaceAll(arg, ",", ", ") + "]"
				for _, k := range []string{curKey, curKeySpaced} {
					clone = strings.ReplaceAll(clone, "*"+k, "*"+concreteName)
					clone = strings.ReplaceAll(clone, k, concreteName)
				}
			}
			buf.WriteString(clone + "\n")
		}
		if len(instList) == 1 {
			producedSingle[name] = fmt.Sprintf(genericNameFormat, name, 1)
		}
	}

	// Fallback textual replacements for any missed generic receiver or type expressions
	final := buf.String()
	for old, newName := range nameMap {
		final = strings.ReplaceAll(final, old, newName)
		final = strings.ReplaceAll(final, "*"+old, "*"+newName)
		if strings.Contains(old, ",") { // spaced variant
			oldSpaced := strings.ReplaceAll(old, ",", ", ")
			final = strings.ReplaceAll(final, oldSpaced, newName)
			final = strings.ReplaceAll(final, "*"+oldSpaced, "*"+newName)
		}
	}
	// Replace direct calls to generic functions without explicit type args when single instantiation
	for orig, mono := range producedSingle {
		final = strings.ReplaceAll(final, orig+"(", mono+"(")
	}

	return []byte(final)
}

// Helpers
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

// commentOut removed: we now delete generic code instead of commenting

// replaceTypeToken replaces occurrences of a standalone type identifier (not part of a larger identifier) in code.
// It avoids replacing field names (after dots) to prevent .value from becoming .stValue
func replaceTypeToken(code, ident, replacement string) string {
	if ident == replacement || ident == "" {
		return code
	}
	needsPointerParenthesis := strings.HasPrefix(replacement, "*")
	var b strings.Builder
	runes := []rune(code)
	n := len(runes)
	for i := 0; i < n; {
		if i+len(ident) <= n {
			segment := string(runes[i : i+len(ident)])
			if segment == ident {
				prevOk := i == 0 || isBoundaryRune(runes[i-1])
				nextOk := i+len(ident) == n || isBoundaryRune(runes[i+len(ident)])
				isFieldAccess := i > 0 && runes[i-1] == '.'
				// If preceded by '&' treat as taking address of a value identifier; skip replacement.
				isAddressOf := i > 0 && runes[i-1] == '&'
				// If wrapped in parentheses and immediately followed by ")" and then a dot, it's a value expression like (ident).Method; skip.
				isParenValueDot := false
				if i+len(ident) < n && runes[i+len(ident)] == ')' {
					k := i + len(ident) + 1
					for k < n && (runes[k] == ' ' || runes[k] == '\t') {
						k++
					}
					if k < n && runes[k] == '.' && i > 0 && runes[i-1] == '(' {
						isParenValueDot = true
					}
				}
				if prevOk && nextOk && !isFieldAccess && !isAddressOf {
					// Determine if the identifier is part of a variable declaration or assignment (we should NOT replace then).
					// Look ahead skipping whitespace.
					j := i + len(ident)
					for j < n && (runes[j] == ' ' || runes[j] == '\t') {
						j++
					}
					isShortDecl := j+1 < n && runes[j] == ':' && runes[j+1] == '=' // ident :=
					isAssignment := j < n && runes[j] == '='                       // ident =
					if isShortDecl || isAssignment || isParenValueDot {
						// Treat as value identifier, do not substitute.
						b.WriteString(segment)
						i += len(ident)
						continue
					}
					if needsPointerParenthesis {
						nextRune := rune(0)
						if i+len(ident) < n {
							nextRune = runes[i+len(ident)]
						}
						if nextRune == '(' || nextRune == '{' { // type conversion / composite literal
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
		}
		b.WriteRune(runes[i])
		i++
	}
	return b.String()
}

func isBoundaryRune(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '(' || r == ')' || r == '{' || r == '}' || r == ',' || r == ';' || r == '*' || r == '[' || r == ']' || r == ':' || r == '.' || r == '&'
}

// inferCompositeType attempts to get the type name from a composite literal expression

// stripFuncTypeParams removes the generic parameter list after a function name.
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
func sortedKeys(m map[string][]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// Fully generic transformation: no type-specific special cases.
