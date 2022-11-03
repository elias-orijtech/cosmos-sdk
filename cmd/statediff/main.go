package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"

	"golang.org/x/tools/go/packages"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "statediff: failed to load packages: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := &packages.Config{
		Mode: packages.NeedImports | packages.NeedSyntax | packages.NeedDeps | packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, "github.com/cosmos/cosmos-sdk/baseapp")
	if err != nil {
		return err
	}
	state := &analyzerState{
		funcs: make(map[*types.Func]bodyInfo),
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
					inf := bodyInfo{decl.Body, pkg.TypesInfo}
					state.funcs[td] = inf
					if rootNames[td.FullName()] {
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
	for _, root := range roots {
		inspect(state, root)
	}
	return nil
}

type bodyInfo struct {
	body *ast.BlockStmt
	info *types.Info
}

type analyzerState struct {
	funcs map[*types.Func]bodyInfo
}

func inspect(state *analyzerState, def *types.Func) {
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
				fmt.Println(t.FullName())
				inspect(state, t)
			}
		}
		return true
	})
}
