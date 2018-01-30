package zmage

import (
	"context"
	"go/build"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func GoLint(ctx context.Context) error {
	mg.CtxDeps(ctx, GoVet)

	pkgs, err := GoPackages(build.Default)
	if err != nil {
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var pkgNames []string
	for _, pkg := range pkgs {
		var dir string
		if dir, err = filepath.Rel(pwd, pkg.Dir); err != nil {
			return err
		}

		pkgNames = append(pkgNames, dir)
	}

	_, err = sh.Exec(nil, os.Stderr, nil, "golint", pkgNames...)
	return err
}
