package main // import "zvelo.io/zapi"

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

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

const name = "zapi"

var (
	version string
	commit  string
	date    string

	app = cli.NewApp()
)

func init() {
	cli.FlagNamePrefixer = flagNamePrefixer

	app.Name = name
	app.Version = fmt.Sprintf("%s (commit %s; built %s; %s)", version, commit, date, runtime.Version())
	app.Usage = "client utility for zvelo api"
	app.EnableBashCompletion = true
	app.BashComplete = complete.Bash
	app.Authors = []cli.Author{
		{Name: "Joshua Rubin", Email: "jrubin@zvelo.com"},
	}

	app.Commands = append(app.Commands,
		complete.BashCommand(complete.Command(name)),
		complete.BashCommand(graphql.Command(name)),
		complete.BashCommand(mock.Command()),
		complete.BashCommand(poll.Command(name)),
		complete.BashCommand(query.Command(name)),
		complete.BashCommand(receiver.Command(name)),
		complete.BashCommand(suggest.Command(name)),
		complete.BashCommand(stream.Command(name)),
		complete.BashCommand(token.Command(name)),
	)
}

func main() {
	if err := app.Run(os.Args); err != nil && err != context.Canceled {
		zvelo.Errorf("%s\n", err)
		os.Exit(1)
	}
}

func flagNamePrefixer(fullName, placeholder string) string {
	var prefixed string
	parts := strings.Split(fullName, ",")

	for i, name := range parts {
		prefixed += "-" + strings.TrimSpace(name)

		if placeholder != "" {
			prefixed += " " + placeholder
		}

		if i < len(parts)-1 {
			prefixed += ", "
		}
	}

	return prefixed
}
