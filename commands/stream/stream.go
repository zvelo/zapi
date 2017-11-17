package stream

import "github.com/urfave/cli"

type cmd struct{}

func Command() cli.Command {
	var c cmd

	return cli.Command{
		Name:  "stream",
		Usage: "stream results from zveloAPI",
		Before: c.setup,
		Action: c.action,
	}
}

func (c *cmd) setup(_ *cli.Context) error {
	return nil
}

func (c *cmd) action(_ *cli.Context) error {
	return nil
}
