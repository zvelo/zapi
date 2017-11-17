package mock

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli"

	"zvelo.io/msg/mock"
)

type cmd struct {
	listen string
}

func (c *cmd) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "listen",
			EnvVar:      "ZVELO_MOCK_LISTEN_ADDRESS",
			Usage:       "address:port for the mock server to listen on ",
			Value:       ":https",
			Destination: &c.listen,
		},
	}
}

func Command() cli.Command {
	var c cmd

	return cli.Command{
		Name:   "mock",
		Usage:  "start a mock zveloAPI server",
		Action: c.action,
		Flags:  c.Flags(),
	}
}

func (c *cmd) action(_ *cli.Context) error {
	fmt.Fprintf(os.Stderr, "mock zveloAPI server listening at: %s\n", c.listen)
	return mock.APIv1().ListenAndServeTLS(context.Background(), c.listen)
}
