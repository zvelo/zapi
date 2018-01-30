package zmage

import (
	"context"

	"github.com/magefile/mage/sh"
)

func Dep(ctx context.Context) error {
	modified, err := Modified("./Gopkg.lock", "./Gopkg.toml")
	if err != nil {
		return err
	}

	if !modified {
		return nil
	}

	if err = sh.Run("dep", "ensure"); err != nil {
		return err
	}

	return Touch("./Gopkg.lock")
}
