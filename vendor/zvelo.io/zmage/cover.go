package zmage

import (
	"bufio"
	"fmt"
	"go/build"
	"io"
	"os"
	"strings"

	"github.com/magefile/mage/sh"
)

func appendCoverage(w io.Writer, fileName string) error {
	c, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = c.Close() }() // #nosec

	scanner := bufio.NewScanner(c)

	for scanner.Scan() {
		text := scanner.Text()

		if strings.HasPrefix(text, "mode:") {
			continue
		}

		if i := strings.Index(text, ":"); i >= 0 {
			file := text[:i]

			if strings.HasSuffix(file, ".pb.gw.go") {
				continue
			}

			if strings.HasSuffix(file, ".pb.go") {
				continue
			}
		}

		_, _ = fmt.Fprintln(w, text) // #nosec
	}

	return scanner.Err()
}

func CoverOnly(flags ...string) error {
	flags = append(flags, "-race")

	pkgs, err := GoPackages(build.Default)
	if err != nil {
		return err
	}

	coverAll, err := os.OpenFile(".coverage-all.out", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = coverAll.Close() }()

	if _, err = coverAll.WriteString("mode: atomic\n"); err != nil {
		return err
	}

	for _, pkg := range pkgs {
		args := []string{"test", "-coverprofile=.coverage.out", "-covermode=atomic"}
		args = append(args, flags...)
		args = append(args, pkg.ImportPath)
		if err = sh.Run(goexe, args...); err != nil {
			return err
		}

		if err = appendCoverage(coverAll, ".coverage.out"); err != nil {
			return err
		}

		if err = os.Remove(".coverage.out"); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func Cover(flags ...string) error {
	if err := CoverOnly(flags...); err != nil {
		return err
	}

	return sh.Run(goexe, "tool", "cover", "-html=.coverage-all.out")
}
