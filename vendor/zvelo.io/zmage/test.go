package zmage

import (
	"context"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func GoTest(ctx context.Context, flags ...string) error {
	mg.CtxDeps(ctx, GoVet)

	flags = append(flags, "-race")

	if err := installTestDeps(flags...); err != nil {
		return err
	}

	var args []string
	testcmd := strings.Split(gotest, " ")

	if len(testcmd) > 1 {
		args = append(args, testcmd[1:]...)
	}

	args = append(args, flags...)
	args = append(args, "./...")
	return sh.RunWith(env, testcmd[0], args...)
}
