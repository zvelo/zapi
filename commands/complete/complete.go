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
	complete(c, c.App.Commands, c.App.VisibleFlags())
}

func BashCommand(cmd cli.Command) cli.Command {
	cmd.SkipArgReorder = true
	cmd.BashComplete = func(c *cli.Context) {
		complete(c, cmd.Subcommands, cmd.VisibleFlags())
	}
	return cmd
}

func flagUsage(flag cli.Flag) string {
	str := flag.String()
	if i := strings.Index(str, "\t"); i >= 0 && len(str) > i+1 {
		return str[i+1:]
	}
	return ""
}

func complete(c *cli.Context, cmds []cli.Command, flags []cli.Flag) {
	var shell shell
	_ = setShell(&shell)

	for _, command := range cmds {
		if command.Hidden {
			continue
		}

		for _, name := range command.Names() {
			switch shell {
			case zsh:
				fmt.Printf("%s:%s\n", name, command.Usage)
			default:
				fmt.Println(name)
			}
		}
	}

	for _, flag := range flags {
		for _, name := range strings.Split(flag.GetName(), ",") {
			if name == FlagName {
				continue
			}

			switch name = strings.TrimSpace(name); len(name) {
			case 0:
				continue
			default:
				name = "-" + name
			}

			switch shell {
			case zsh:
				fmt.Printf("%s:%s\n", name, flagUsage(flag))
			default:
				fmt.Println(name)
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

func setShell(s *shell) error {
	switch shell := filepath.Base(os.Getenv("SHELL")); shell {
	case "bash":
		*s = bash
	case "zsh":
		*s = zsh
	default:
		return errors.Errorf("unsupported shell: %s", shell)
	}
	return nil
}

func (c *cmd) setup(_ *cli.Context) error {
	return setShell(&c.Shell)
}

func (c *cmd) run(_ *cli.Context) error {
	if c.Shell == bash {
		return bashCompleteTpl.Execute(os.Stdout, c)
	}

	return zshCompleteTpl.Execute(os.Stdout, c)
}

var bashCompleteTpl = template.Must(template.New("bashShellFunc").Parse(bashShellFunc))

var bashShellFunc = `#!/bin/bash

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

var zshCompleteTpl = template.Must(template.New("zshShellFunc").Parse(zshShellFunc))

var zshShellFunc = `_{{ .AppName }}_autocomplete() {
  local -a opts
  opts=("${(@f)$(${words[@]:0:#words[@]-1} --{{ .FlagName }})}")

  _describe 'values' opts

  return
}

compdef _{{ .AppName }}_autocomplete {{ .AppName }}
`
