package zmage

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var goPackageData = struct {
	data []*build.Package
	err  error
	once sync.Once
}{}

func GoPackages(ctx build.Context) ([]*build.Package, error) {
	goPackageData.once.Do(func() {
		pwd, err := os.Getwd()
		if err != nil {
			goPackageData.err = err
			return
		}

		goPackageData.err = filepath.Walk(pwd, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				return nil
			}

			base := filepath.Base(path)

			if base == vendor {
				return filepath.SkipDir
			}

			if strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}

			if strings.HasPrefix(base, "_") {
				return filepath.SkipDir
			}

			matches, err := filepath.Glob(filepath.Join(path, "*.go"))
			if err != nil {
				return err
			}

			if len(matches) == 0 {
				return nil
			}

			pkg, err := ctx.ImportDir(path, 0)
			if err != nil {
				return err
			}

			goPackageData.data = append(goPackageData.data, pkg)

			return nil
		})
	})

	return goPackageData.data, goPackageData.err
}
