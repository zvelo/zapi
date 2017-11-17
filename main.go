package main // import "zvelo.io/zapi"

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/urfave/cli"

	"zvelo.io/zapi/commands/complete"
	"zvelo.io/zapi/commands/graphql"
	"zvelo.io/zapi/commands/mock"
	"zvelo.io/zapi/commands/poll"
	"zvelo.io/zapi/commands/query"
	"zvelo.io/zapi/commands/stream"
	"zvelo.io/zapi/commands/suggest"
	"zvelo.io/zapi/commands/token"
	"zvelo.io/zapi/internal/zvelo"
)

const appName = "zapi"

var (
	version = "v1.4.0"
	app     = cli.NewApp()
)

func init() {
	cli.BashCompletionFlag = cli.BoolFlag{
		Name:   complete.FlagName,
		Hidden: true,
	}

	app.Name = appName
	app.Version = fmt.Sprintf("%s (%s)", version, runtime.Version())
	app.Usage = "client utility for zvelo api"
	app.EnableBashCompletion = true
	app.BashComplete = complete.Bash
	app.Authors = []cli.Author{
		{Name: "Joshua Rubin", Email: "jrubin@zvelo.com"},
	}

	app.Commands = append(app.Commands,
		complete.Command(appName),
		graphql.Command(appName),
		mock.Command(),
		poll.Command(appName),
		query.Command(appName),
		suggest.Command(),
		stream.Command(appName),
		token.Command(appName),
	)

	for _, cmd := range app.Commands {
		cmd.BashComplete = complete.BashCommand(cmd)
	}
}

func main() {
	if err := app.Run(os.Args); err != nil && err != context.Canceled {
		zvelo.Errorf("%s\n", err)
		os.Exit(1)
	}
}
