package complete

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const FlagName = "compgen"

func init() {
	cli.BashCompletionFlag = cli.BoolFlag{Name: FlagName}
}

func Command(appName string) cli.Command {
	c := cmd{
		AppName:  appName,
		FlagName: FlagName,
		Bash:     bash,
		Zsh:      zsh,
	}

	return cli.Command{
		Name:        "complete",
		Usage:       "generate autocomplete script",
		Description: "eval \"$(" + appName + " complete)\"",
		Before:      c.setup,
		Action:      c.run,
	}
}

func Bash(c *cli.Context) {
	complete(c, c.App.Commands, c.App.Flags)
}

func BashCommand(cmd cli.Command) cli.Command {
	cmd.BashComplete = func(c *cli.Context) {
		complete(c, cmd.Subcommands, cmd.Flags)
	}
	return cmd
}

func complete(c *cli.Context, cmds []cli.Command, flags []cli.Flag) {
	for _, command := range cmds {
		if command.Hidden {
			continue
		}

		for _, name := range command.Names() {
			fmt.Println(name)
		}
	}

	for _, flag := range flags {
		for _, name := range strings.Split(flag.GetName(), ",") {
			if name == FlagName {
				continue
			}

			switch name = strings.TrimSpace(name); len(name) {
			case 0:
			case 1:
				fmt.Println("-" + name)
			default:
				fmt.Println("--" + name)
			}
		}
	}
}

type shell int

const (
	bash shell = iota
	zsh
)

type cmd struct {
	AppName  string
	FlagName string
	Shell    shell
	Bash     shell
	Zsh      shell
}

func (c *cmd) setup(_ *cli.Context) error {
	switch shell := filepath.Base(os.Getenv("SHELL")); shell {
	case "bash":
		c.Shell = bash
	case "zsh":
		c.Shell = zsh
	default:
		return errors.Errorf("unsupported shell: %s", shell)
	}
	return nil
}

func (c *cmd) run(_ *cli.Context) error {
	return completeTpl.Execute(os.Stdout, c)
}

var completeTpl = template.Must(template.New("shellFunc").Parse(shellFunc))

var shellFunc = `{{ if eq .Shell .Bash }}#!/bin/bash{{ end }}{{ if eq .Shell .Zsh }}autoload -U compinit && compinit
autoload -U bashcompinit && bashcompinit{{ end }}

_{{ .AppName }}_autocomplete() {
     local cur opts base
     COMPREPLY=()
     cur="${COMP_WORDS[COMP_CWORD]}"
     opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --{{ .FlagName }} )
     COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
     return 0
 }

 complete -F _{{ .AppName }}_autocomplete {{ .AppName }}
`
