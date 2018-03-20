package zmage

import (
	"bufio"
	"go/build"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/magefile/mage/sh"
)

var protoData = struct {
	data map[string][]string
	err  error
	once sync.Once
}{}

func protoFiles() (map[string][]string, error) {
	protoData.once.Do(func() {
		protoData.err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
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

				return nil
			}

			if filepath.Ext(path) != ".proto" {
				return nil
			}

			if protoData.data == nil {
				protoData.data = map[string][]string{}
			}

			dir := filepath.Dir(path)

			protoData.data[dir] = append(protoData.data[dir], path)

			return nil
		})
	})

	return protoData.data, protoData.err
}

var protoBuildLock sync.Mutex

func protoBuildOne(preFn func() error, cmd, gwPkgDir string, files, args []string) error {
	protoBuildLock.Lock()
	defer protoBuildLock.Unlock()

	if err := os.RemoveAll("../../zvelo"); err != nil {
		return err
	}

	if err := os.Symlink("zvelo.io", "../../zvelo"); err != nil {
		return err
	}

	args = append(args,
		"-I../..",
		"-I"+filepath.Join(gwPkgDir, "../third_party/googleapis"),
	)

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	for _, file := range files {
		args = append(args, filepath.Join("zvelo", filepath.Base(pwd), file))
	}

	if preFn != nil {
		if err = preFn(); err != nil {
			return err
		}
	}

	if err = sh.Run(cmd, args...); err != nil {
		return err
	}

	return os.RemoveAll("../../zvelo")
}

func protoBuild(exts []string, useFileFn func(string) (bool, error), preFn func() error, cmd string, args ...string) ([]string, error) {
	dirFiles, err := protoFiles()
	if err != nil {
		return nil, err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	gwPkg, err := build.Import("github.com/grpc-ecosystem/grpc-gateway/runtime", pwd, 0)
	if err != nil {
		return nil, err
	}

	var updatedFiles []string

	for _, files := range dirFiles {
		newFiles := files

		if useFileFn != nil {
			newFiles = nil

			for _, file := range files {
				var use bool
				if use, err = useFileFn(file); err != nil {
					return nil, err
				}

				if use {
					newFiles = append(newFiles, file)
				}
			}
		}

		var modified bool
		for _, file := range newFiles {
			for _, ext := range exts {
				pbFile := strings.Replace(file, ".proto", ext, -1)

				var fileModified bool
				if fileModified, err = Modified(pbFile, file); err != nil {
					return nil, err
				}

				if fileModified {
					modified = true
					break
				}
			}
		}

		if !modified {
			continue
		}

		if err = protoBuildOne(preFn, cmd, gwPkg.Dir, newFiles, args); err != nil {
			return nil, err
		}

		updatedFiles = append(updatedFiles, newFiles...)
	}

	return updatedFiles, nil
}

var gogoZvelo onceData

func installGogoZvelo() error {
	gogoZvelo.once.Do(func() {
		gogoZvelo.err = InstallExe(build.Default, "zvelo.io/msg/cmd/protoc-gen-gozvelo")
	})
	return gogoZvelo.err
}

func ProtoGo() ([]string, error) {
	return protoBuild([]string{".pb.go"}, nil, installGogoZvelo, protoc, "--gozvelo_out=Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,plugins=grpc:../..")
}

var serviceRe = regexp.MustCompile(`^\s*option\s+\(google\.api\.http\)\s+=\s+{$`)

func protoUsesGRPCGateway(file string) (bool, error) {
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if serviceRe.Match(scanner.Bytes()) {
			return true, nil
		}
	}

	return false, scanner.Err()
}

func ProtoGRPCGateway() ([]string, error) {
	return protoBuild([]string{".pb.gw.go"}, protoUsesGRPCGateway, nil, protoc, "--grpc-gateway_out=logtostderr=true,request_context=true:../..")
}

func ProtoSwagger() ([]string, error) {
	return protoBuild([]string{".swagger.json"}, protoUsesGRPCGateway, nil, protoc, "--swagger_out=logtostderr=true:../..")
}

func ProtoPython() ([]string, error) {
	return protoBuild([]string{"_pb2.py", "_pb2_grpc.py"}, nil, nil, python,
		"-m", "grpc_tools.protoc",
		"--python_out=../..",
		"--grpc_python_out=../..",
	)
}

func protoUsesFile(name string) func(string) (bool, error) {
	return func(file string) (bool, error) {
		return name == file, nil
	}
}

func Protoset(name string) ([]string, error) {
	ext := ".protoset"
	return protoBuild([]string{ext}, protoUsesFile(name), nil, protoc,
		"--descriptor_set_out="+strings.Replace(name, ".proto", ext, -1),
		"--include_imports",
	)
}
