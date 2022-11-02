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
		Mode: packages.NeedImports | packages.NeedSyntax | packages.NeedDeps | packages.NeedName | packages.NeedTypesInfo | packages.NeedTypes,
	}
	pkgs, err := packages.Load(cfg, "github.com/cosmos/cosmos-sdk/baseapp")
	if err != nil {
		return err
	}
	state := &analyzerState{
		stateFuncs: map[string]bool{
			"(*github.com/cosmos/cosmos-sdk/baseapp.BaseApp).DeliverTx": true,
		},
		funcs: make(map[string]pkgFunc),
	}
	imported := make(map[*packages.Package]bool)
	var addPkg func(pkg *packages.Package)
	addPkg = func(pkg *packages.Package) {
		if imported[pkg] {
			return
		}
		imported[pkg] = true
		fmt.Println(pkg.Name)
		for _, f := range pkg.Syntax {
			for _, decl := range f.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
					td := pkg.TypesInfo.Defs[decl.Name].(*types.Func)
					state.funcs[td.FullName()] = pkgFunc{pkg, decl}
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
	for name := range state.stateFuncs {
		decl, ok := state.funcs[name]
		if !ok {
			return fmt.Errorf("statediff: state function %s not found", name)
		}
		inspect(state, decl)
	}
	return nil
}

type pkgFunc struct {
	pkg *packages.Package
	fun *ast.FuncDecl
}

type analyzerState struct {
	stateFuncs map[string]bool
	funcs      map[string]pkgFunc
}

func inspect(state *analyzerState, decl pkgFunc) {
	ast.Inspect(decl.fun.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			var id *ast.Ident
			switch fun := n.Fun.(type) {
			case *ast.Ident:
				id = fun
			case *ast.SelectorExpr:
				id = fun.Sel
			}
			switch t := decl.pkg.TypesInfo.Uses[id].(type) {
			case *types.Func:
				fmt.Printf("uses type %T, id %v: %v full name %s\n", t, t.Id(), n, t.FullName())
			}
		}
		return true
	})
}
