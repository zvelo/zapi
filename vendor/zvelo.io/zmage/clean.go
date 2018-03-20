package zmage

import (
	"os"
	"strings"
)

func Clean(files ...string) error {
	files = append(files,
		"./.coverage.out",
		"./.coverage-all.out",
	)

	dirFiles, err := protoFiles()
	if err != nil {
		return err
	}

	for _, df := range dirFiles {
		for _, file := range df {
			for _, ext := range []string{".pb.go", ".pb.gw.go", "_pb2.py", "_pb2_grpc.py", ".swagger.json", ".protoset"} {
				pbFile := strings.Replace(file, ".proto", ext, -1)
				files = append(files, pbFile)
			}
		}
	}

	for _, file := range files {
		if err := os.RemoveAll(file); err != nil {
			return err
		}
	}

	return nil
}
