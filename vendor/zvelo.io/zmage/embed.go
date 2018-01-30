package zmage

import (
	"errors"

	"github.com/mjibson/esc/embed"
)

type EmbedConfig = embed.Config

func Embed(conf EmbedConfig) error {
	if conf.Package == "" {
		conf.Package = "main"
	}

	if conf.OutputFile == "" {
		panic(errors.New("OutputFile is required"))
	}

	return embed.Run(&conf)
}
