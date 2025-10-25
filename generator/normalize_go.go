package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/printer"
	"go/scanner"
	"go/token"
	"unicode"

	"golang.org/x/tools/imports"
)

// TryStripLayoutNewlinesAndCheck reports whether removing layout newlines
// (i.e., joining tokens where safe; preserving newlines inside literals/comments)
// yields exactly the same program under parsing/printing normalization.
func TryStripLayoutNewlinesAndCheck(src []byte) (same bool, minified []byte, err error) {
	minified, err = minifyNoLayoutNewlines(src)
	if err != nil {
		return false, nil, err
	}
	normA, err := normalizeGo(src)
	if err != nil {
		return false, nil, fmt.Errorf("normalize original: %w", err)
	}
	normB, err := normalizeGo(minified)
	if err != nil {
		return false, nil, fmt.Errorf("normalize minified: %w", err)
	}
	return bytes.Equal(normA, normB), minified, nil
}

// minifyNoLayoutNewlines re-tokenizes src and re-emits tokens without layout
// newlines. It preserves token text for strings/comments (so their embedded
// newlines remain). Between tokens, it emits only the minimal spaces required
// to avoid token merging (e.g., "if"+"(" needs no space; "var"+"x" needs space).
func minifyNoLayoutNewlines(src []byte) ([]byte, error) {
	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("in.go", -1, len(src))
	// Let the scanner do semicolon insertion as usual; we only re-emit tokens.
	s.Init(file, src, nil, scanner.ScanComments)

	var out bytes.Buffer
	var prevTok token.Token
	var prevText string
	first := true

	emit := func(sp string) {
		out.WriteString(sp)
	}

	needsSpace := func(aTok token.Token, aText string, bTok token.Token, bText string) bool {
		// Rule of thumb: put space if omitting space would join two identifiers/keywords,
		// numbers, or make an ambiguous operator sequence.
		// We approximate by looking at rune classes at the join.
		if aTok == token.SEMICOLON || aTok == token.COMMENT {
			// comments already carry their own spacing; after comments, ensure a space
			// if next token starts with an identifier/number.
			if len(bText) > 0 && (isIdentStart(rune(bText[0])) || unicode.IsDigit(rune(bText[0]))) {
				return true
			}
			return false
		}
		if aTok.IsKeyword() || aTok == token.IDENT {
			// IDENT/keyword followed by IDENT/keyword/NUMBER needs a space.
			if bTok.IsKeyword() || bTok == token.IDENT || bTok == token.INT || bTok == token.FLOAT || bTok == token.IMAG || bTok == token.CHAR || bTok == token.STRING {
				return true
			}
		}
		// Number followed by ident/number needs a space.
		if aTok == token.INT || aTok == token.FLOAT || aTok == token.IMAG {
			if bTok == token.IDENT || bTok == token.INT || bTok == token.FLOAT || bTok == token.IMAG {
				return true
			}
		}
		// Two dots: avoid ".." (not a token); but "." followed by IDENT is selector -> no space.
		if aTok == token.PERIOD && bTok == token.PERIOD {
			return true
		}
		// Operators that would glue awkwardly, e.g., "<" + "<" should be fine (becomes "<<"),
		// but if that changes meaning we rely on parser equivalence check later.
		return false
	}

	for {
		pos, tok, lit := s.Scan()
		_ = pos
		if tok == token.EOF {
			break
		}
		text := lit
		if text == "" {
			text = tok.String()
		}

		// For comments and string/char literals, preserve exact token text (may include newlines).
		if !first {
			if needsSpace(prevTok, prevText, tok, text) {
				emit(" ")
			}
		}
		emit(text)
		prevTok, prevText = tok, text
		first = false
	}
	return out.Bytes(), nil
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

// normalizeGo: parse, drop comments, print deterministically (like gofmt).
func normalizeGo(src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	// Parse with comments so we can drop them explicitly for deterministic output.
	file, err := parser.ParseFile(fset, "in.go", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	// Strip comments.
	file.Comments = nil
	// Print with canonical settings.
	var buf bytes.Buffer
	cfg := &printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	if err := cfg.Fprint(&buf, fset, file); err != nil {
		return nil, err
	}

	src, err = format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return imports.Process("", src, nil)
}
