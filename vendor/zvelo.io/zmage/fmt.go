package zmage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/magefile/mage/sh"
)

var goFileData = struct {
	data []string
	err  error
	once sync.Once
}{}

func goFiles() ([]string, error) {
	goFileData.err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if path == "." {
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
		}

		if filepath.Ext(path) == ".go" {
			goFileData.data = append(goFileData.data, path)
		}

		return nil
	})

	return goFileData.data, goFileData.err
}

func GoFmt() error {
	files, err := goFiles()
	if err != nil {
		return err
	}

	var fmtFiles []string
	for _, file := range files {
		if strings.HasSuffix(file, ".pb.go") {
			continue
		}
		fmtFiles = append(fmtFiles, file)
	}

	s, err := sh.Output("gofmt", append([]string{"-l", "-s"}, fmtFiles...)...)
	if err != nil {
		return err
	}

	if s != "" {
		fmt.Fprintln(os.Stderr, "The following files need `gofmt -s`:")
		fmt.Fprintln(os.Stderr, s)
		return fmt.Errorf("improperly formatted go files")
	}

	return nil
}
