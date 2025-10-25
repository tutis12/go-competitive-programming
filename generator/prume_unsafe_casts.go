package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

func PruneUnsafeCasts(src []byte) []byte {
	// Requires: "bytes", "go/ast", "go/parser", "go/printer", "go/token", "strings"
	// (sanitizeCode is assumed to exist in your codebase, same as before.)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "merged.go", src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return src
	}

	// Build parent map so we can replace child nodes in-place.
	parent := map[ast.Node]ast.Node{}
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		ast.Inspect(n, func(ch ast.Node) bool {
			if ch == nil {
				return false
			}
			if ch != n {
				if _, seen := parent[ch]; !seen {
					parent[ch] = n
				}
			}
			return true
		})
		return false
	})

	// Helper: replace old expression node with new in its parent.
	setChildExprInParent := func(p ast.Node, old, newE ast.Expr) bool {
		switch x := p.(type) {
		case *ast.ExprStmt:
			if x.X == old {
				x.X = newE
				return true
			}
		case *ast.ReturnStmt:
			for i := range x.Results {
				if x.Results[i] == old {
					x.Results[i] = newE
					return true
				}
			}
		case *ast.AssignStmt:
			for i := range x.Lhs {
				if x.Lhs[i] == old {
					x.Lhs[i] = newE
					return true
				}
			}
			for i := range x.Rhs {
				if x.Rhs[i] == old {
					x.Rhs[i] = newE
					return true
				}
			}
		case *ast.CallExpr:
			if x.Fun == old {
				x.Fun = newE
				return true
			}
			for i := range x.Args {
				if x.Args[i] == old {
					x.Args[i] = newE
					return true
				}
			}
		case *ast.SelectorExpr:
			if x.X == old {
				x.X = newE
				return true
			}
		case *ast.ParenExpr:
			if x.X == old {
				x.X = newE
				return true
			}
		case *ast.StarExpr:
			if x.X == old {
				x.X = newE
				return true
			}
		case *ast.UnaryExpr:
			if x.X == old {
				x.X = newE
				return true
			}
		case *ast.BinaryExpr:
			if x.X == old {
				x.X = newE
				return true
			}
			if x.Y == old {
				x.Y = newE
				return true
			}
		case *ast.IndexExpr:
			if x.X == old {
				x.X = newE
				return true
			}
			if x.Index == old {
				x.Index = newE
				return true
			}
		case *ast.IndexListExpr:
			if x.X == old {
				x.X = newE
				return true
			}
			for i := range x.Indices {
				if x.Indices[i] == old {
					x.Indices[i] = newE
					return true
				}
			}
		case *ast.SliceExpr:
			if x.X == old {
				x.X = newE
				return true
			}
			if x.Low == old {
				x.Low = newE
				return true
			}
			if x.High == old {
				x.High = newE
				return true
			}
			if x.Max == old {
				x.Max = newE
				return true
			}
		case *ast.TypeAssertExpr:
			if x.X == old {
				x.X = newE
				return true
			}
		case *ast.KeyValueExpr:
			if x.Key == old {
				x.Key = newE
				return true
			}
			if x.Value == old {
				x.Value = newE
				return true
			}
		case *ast.CompositeLit:
			if x.Type == old {
				x.Type = newE
				return true
			}
			for i := range x.Elts {
				if x.Elts[i] == old {
					x.Elts[i] = newE
					return true
				}
			}
		case *ast.ValueSpec:
			if x.Type == old {
				x.Type = newE
				return true
			}
			for i := range x.Values {
				if x.Values[i] == old {
					x.Values[i] = newE
					return true
				}
			}
		case *ast.Field:
			if x.Type == old {
				x.Type = newE
				return true
			}
		}
		return false
	}

	// Matches: *(*T)(unsafe.Pointer(&x))  ->  T(x)
	matchAndRewrite := func(node ast.Node) bool {
		u, ok := node.(*ast.UnaryExpr)
		if !ok || u.Op != token.MUL {
			return false
		}
		// Expect: *( <paren or direct> )
		pe, ok := u.X.(*ast.ParenExpr)
		var inner ast.Expr
		if ok {
			inner = pe.X
		} else {
			inner = u.X
		}

		// Expect: (*T)( unsafe.Pointer( &x ) )
		ce, ok := inner.(*ast.CallExpr)
		if !ok {
			return false
		}
		// Fun should be (*T) — i.e., a parenthesized pointer type
		parT, ok := ce.Fun.(*ast.ParenExpr)
		if !ok {
			return false
		}
		ptrT, ok := parT.X.(*ast.StarExpr)
		if !ok {
			return false
		}
		// Arg should be unsafe.Pointer(&x)
		if len(ce.Args) != 1 {
			return false
		}
		argCall, ok := ce.Args[0].(*ast.CallExpr)
		if !ok {
			return false
		}
		se, ok := argCall.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		pkgIdent, ok := se.X.(*ast.Ident)
		if !ok || pkgIdent.Name != "unsafe" || se.Sel == nil || se.Sel.Name != "Pointer" {
			return false
		}
		if len(argCall.Args) != 1 {
			return false
		}
		addr, ok := argCall.Args[0].(*ast.UnaryExpr)
		if !ok || addr.Op != token.AND {
			return false
		}
		// addr.X is the original value expression "x".
		origVal := addr.X

		// We want T(x) where T is the non-pointer element of *T.
		targetType := ptrT.X // ast.Expr of T
		newExpr := &ast.CallExpr{
			Fun:  targetType,
			Args: []ast.Expr{origVal},
		}

		// Replace in parent
		p := parent[u]
		if p == nil {
			return false
		}
		return setChildExprInParent(p, u, newExpr)
	}

	// Walk and rewrite every matching node; iterate until fixed point in case nested.
	changed := true
	for changed {
		changed = false
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				return true
			}
			if matchAndRewrite(n) {
				changed = true
				return false
			}
			return true
		})
		// Rebuild parent map if changed
		if changed {
			parent = map[ast.Node]ast.Node{}
			ast.Inspect(file, func(n ast.Node) bool {
				if n == nil {
					return true
				}
				ast.Inspect(n, func(ch ast.Node) bool {
					if ch == nil {
						return false
					}
					if ch != n {
						if _, seen := parent[ch]; !seen {
							parent[ch] = n
						}
					}
					return true
				})
				return false
			})
		}
	}

	// If "unsafe" is now unused, drop it from imports.
	usesUnsafe := false
	ast.Inspect(file, func(n ast.Node) bool {
		se, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := se.X.(*ast.Ident); ok && id.Name == "unsafe" {
			usesUnsafe = true
			return false
		}
		return true
	})
	if !usesUnsafe {
		// Filter import specs that reference "unsafe".
		var newDecls []ast.Decl
		for _, d := range file.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.IMPORT {
				newDecls = append(newDecls, d)
				continue
			}
			var kept []ast.Spec
			for _, sp := range gd.Specs {
				isUnsafe := false
				if im, ok := sp.(*ast.ImportSpec); ok {
					if im.Path != nil && strings.Trim(im.Path.Value, `"`) == "unsafe" {
						isUnsafe = true
					}
				}
				if !isUnsafe {
					kept = append(kept, sp)
				}
			}
			if len(kept) > 0 {
				ng := *gd
				ng.Specs = kept
				newDecls = append(newDecls, &ng)
			}
			// else: drop entire empty import decl
		}
		file.Decls = newDecls
	}

	// Pretty-print the modified file, then run your existing sanitize for final cleanup.
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, file); err != nil {
		return src
	}
	return buf.Bytes()
}
