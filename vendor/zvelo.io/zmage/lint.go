package zmage

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
)

var (
	lintDeadline = 30 * time.Second

	disabledLinters []string

	enabledLinters = []string{
		"deadcode",
		"errcheck",
		"gas",
		"goconst",
		"gofmt",
		"goimports",
		"golint",
		"gotype",
		"gotypex",
		"ineffassign",
		"interfacer",
		"megacheck",
		"misspell",
		"unconvert",
		"varcheck",
		"vet",
		"vetshadow",
	}

	lintIgnoreSuffixes = []string{
		".pb.go",
		".pb.gw.go",
		"_string.go",
		"bindata.go",
		"bindata_assetfs.go",
		"static.go",
	}
)

func init() {
	if val := os.Getenv("GO_METALINTER_DEADLINE"); val != "" {
		if dur, err := time.ParseDuration(val); err == nil {
			lintDeadline = dur
		}
	}

	if val := os.Getenv("GO_METALINTER_ENABLED"); val != "" {
		enabledLinters = strings.Fields(val)
	}

	if val := os.Getenv("GO_METALINTER_DISABLED"); val != "" {
		disabledLinters = strings.Fields(val)
	}

	if val := os.Getenv("GO_METALINTER_IGNORE_SUFFIXES"); val != "" {
		lintIgnoreSuffixes = strings.Fields(val)
	}
}

func GoLint(ctx context.Context) error {
	pkgs, err := GoPackages(build.Default)
	if err != nil {
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	args := []string{
		"--disable-all",
		"--tests",
		"--deadline=" + lintDeadline.String(),
	}

	for _, val := range enabledLinters {
		args = append(args, "--enable="+val)
	}

	for _, val := range disabledLinters {
		args = append(args, "--disable="+val)
	}

	for _, pkg := range pkgs {
		var dir string
		if dir, err = filepath.Rel(pwd, pkg.Dir); err != nil {
			return err
		}

		args = append(args, dir)
	}

	pr, pw := io.Pipe()
	go lintReadResult(os.Stderr, pr)
	_, err = sh.Exec(nil, pw, nil, "gometalinter", args...)
	return err
}

var (
	levelRe    = regexp.MustCompile(`\A([^:]*):(\d*):(\d*):(\w+): (.*?) \((\w+)\)\z`)
	commentRe  = regexp.MustCompile(` should have comment.* or be unexported`)
	mageDeadRe = regexp.MustCompile(`\A[A-Z]\w* is unused\z`)
)

type lintPart int

// The different fields of the gometalinter line
const (
	lintFile lintPart = 1 + iota
	lintLine
	lintColumn
	lintLevel
	lintMessage
	lintLinter
)

func lintHasSuffix(m [][]byte) bool {
	for _, is := range lintIgnoreSuffixes {
		if bytes.HasSuffix(m[lintFile], []byte(is)) {
			return true
		}
	}
	return false
}

func lintMsgMissingComment(m [][]byte) bool {
	if string(m[lintLinter]) != "golint" {
		return false
	}

	return commentRe.Match(m[lintMessage])
}

func lintMagefileDeadcode(m [][]byte) bool {
	if string(m[lintLinter]) != "deadcode" {
		return false
	}

	if !bytes.HasSuffix(m[lintFile], []byte("magefile.go")) {
		return false
	}

	return mageDeadRe.Match(m[lintMessage])
}

func lintReadResult(w io.Writer, r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		if m := levelRe.FindSubmatch(scanner.Bytes()); m != nil {
			if lintHasSuffix(m) ||
				lintMsgMissingComment(m) ||
				lintMagefileDeadcode(m) {
				continue
			}
		}

		fmt.Fprintln(w, scanner.Text())
	}
}
