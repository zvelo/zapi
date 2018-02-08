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
	"zvelo.io/zapi/commands/receiver"
	"zvelo.io/zapi/commands/stream"
	"zvelo.io/zapi/commands/suggest"
	"zvelo.io/zapi/commands/token"
	"zvelo.io/zapi/internal/zvelo"
)

const appName = "zapi"

var (
	version = "v1.6.0"
	app     = cli.NewApp()
)

func init() {
	app.Name = appName
	app.Version = fmt.Sprintf("%s (%s)", version, runtime.Version())
	app.Usage = "client utility for zvelo api"
	app.EnableBashCompletion = true
	app.BashComplete = complete.Bash
	app.Authors = []cli.Author{
		{Name: "Joshua Rubin", Email: "jrubin@zvelo.com"},
	}

	app.Commands = append(app.Commands,
		complete.BashCommand(complete.Command(appName)),
		complete.BashCommand(graphql.Command(appName)),
		complete.BashCommand(mock.Command()),
		complete.BashCommand(poll.Command(appName)),
		complete.BashCommand(query.Command(appName)),
		complete.BashCommand(receiver.Command(appName)),
		complete.BashCommand(suggest.Command(appName)),
		complete.BashCommand(stream.Command(appName)),
		complete.BashCommand(token.Command(appName)),
	)
}

func main() {
	if err := app.Run(os.Args); err != nil && err != context.Canceled {
		zvelo.Errorf("%s\n", err)
		os.Exit(1)
	}
}
