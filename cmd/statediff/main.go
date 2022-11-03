package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sourcegraph/go-diff/diff"
	"golang.org/x/tools/go/packages"
)

var dir = flag.String("dir", ".", "base directory for the patch")

func main() {
	flag.Parse()
	*dir, _ = filepath.Abs(*dir)
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "statediff: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fset := new(token.FileSet)
	cfg := &packages.Config{
		Fset: fset,
		Mode: packages.NeedImports | packages.NeedSyntax | packages.NeedDeps | packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, "github.com/cosmos/cosmos-sdk/baseapp")
	if err != nil {
		return err
	}
	patch, err := parsePatch(os.Stdin)
	if err != nil {
		return err
	}
	state := &analyzerState{
		fset:  fset,
		funcs: make(map[*types.Func]BodyInfo),
	}
	imported := make(map[*packages.Package]bool)
	rootNames := map[string]bool{
		"(*github.com/cosmos/cosmos-sdk/baseapp.BaseApp).DeliverTx": true,
	}
	var roots []*types.Func
	var addPkg func(pkg *packages.Package)
	addPkg = func(pkg *packages.Package) {
		if imported[pkg] {
			return
		}
		imported[pkg] = true
		for _, f := range pkg.Syntax {
			for _, decl := range f.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
					td := pkg.TypesInfo.Defs[decl.Name].(*types.Func)
					inf := BodyInfo{decl.Body, pkg.TypesInfo}
					state.funcs[td] = inf
					if fn := td.FullName(); rootNames[fn] {
						delete(rootNames, fn)
						roots = append(roots, td)
					}
				}
			}
		}
		for _, pkg := range pkg.Imports {
			addPkg(pkg)
		}
	}
	for _, pkg := range pkgs {
		addPkg(pkg)
	}
	var missing []string
	for n := range rootNames {
		missing = append(missing, n)
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing roots: %v", strings.Join(missing, ","))
	}
	for _, root := range roots {
		inspect(state, patch, root)
	}
	return nil
}

func parsePatch(r io.Reader) (Patch, error) {
	diffs := diff.NewMultiFileDiffReader(os.Stdin)
	var p Patch
	for {
		d, err := diffs.ReadFile()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return Patch{}, fmt.Errorf("failed to read diff: %v", err)
		}
		for _, hunk := range d.Hunks {
			startLine := int(hunk.OrigStartLine)
			p = append(p, Hunk{
				file:      filepath.Join(*dir, d.OrigName),
				startLine: startLine,
				endLine:   startLine + int(hunk.OrigLines),
			})
		}
	}
	sort.Slice(p, func(i, j int) bool {
		h1, h2 := p[i], p[j]
		switch strings.Compare(h1.file, h2.file) {
		case -1:
			return true
		case +1:
			return false
		default:
			return h1.startLine <= h2.startLine
		}
	})
	return p, nil
}

// Patch is a slice of Hunks, sorted by path then starting line to
// make Edits efficient.
type Patch []Hunk

type Hunk struct {
	file      string
	startLine int
	endLine   int
}

// Edits reports whether the patch edits the file in the line range
// specified. The range is inclusive.
func (p Patch) Edits(file string, startLine, endLine int) bool {
	idx := sort.Search(len(p), func(i int) bool {
		h := p[i]
		switch strings.Compare(file, h.file) {
		case -1:
			return true
		case +1:
			return false
		default:
			return startLine <= h.endLine
		}
	})
	for i := idx; i < len(p); i++ {
		h := p[i]
		if h.file != file {
			return false
		}
		if h.startLine <= endLine {
			return true
		}
	}
	return false
}

type BodyInfo struct {
	body *ast.BlockStmt
	info *types.Info
}

type analyzerState struct {
	fset  *token.FileSet
	funcs map[*types.Func]BodyInfo
}

func inspect(state *analyzerState, patch Patch, def *types.Func) {
	inf, ok := state.funcs[def]
	if !ok || inf.body == nil {
		return
	}
	delete(state.funcs, def)
	ast.Inspect(inf.body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			var id *ast.Ident
			switch fun := n.Fun.(type) {
			case *ast.Ident:
				id = fun
			case *ast.SelectorExpr:
				id = fun.Sel
			}
			switch t := inf.info.Uses[id].(type) {
			case *types.Func:
				file := state.fset.File(id.Pos())
				start := state.fset.PositionFor(id.Pos(), false)
				end := state.fset.PositionFor(id.End(), false)
				if start.IsValid() && end.IsValid() && patch.Edits(start.Filename, start.Line, end.Line) {
					fmt.Println("edit!", file, start, end)
				}
				inspect(state, patch, t)
			}
		}
		return true
	})
}
