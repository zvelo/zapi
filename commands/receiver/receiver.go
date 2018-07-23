package receiver

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/urfave/cli"
	"zvelo.io/go-zapi/callback"
	"zvelo.io/httpsig"
	msg "zvelo.io/msg/msgpb"
	"zvelo.io/zapi/internal/zvelo"
	"zvelo.io/zapi/results"
)

type cmd struct {
	appName            string
	listen             string
	debug, json        bool
	callbackNoValidate bool
	callbackNoKeyCache bool
	keyGetter          httpsig.KeyGetter
}

func (c *cmd) Flags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:        "debug",
			EnvVar:      "ZVELO_DEBUG",
			Usage:       "enable debug logging",
			Destination: &c.debug,
		},
		cli.BoolFlag{
			Name:        "json",
			EnvVar:      "ZVELO_JSON",
			Usage:       "Print raw JSON response",
			Destination: &c.json,
		},
		cli.StringFlag{
			Name:        "listen",
			EnvVar:      "ZVELO_RECEIVER_LISTEN_ADDRESS",
			Usage:       "address and port to listen for callbacks",
			Value:       ":8080",
			Destination: &c.listen,
		},
		cli.BoolFlag{
			Name:        "no-validate-callback",
			EnvVar:      "ZVELO_NO_VALIDATE_CALLBACK",
			Usage:       "do not validate callback signatures",
			Destination: &c.callbackNoValidate,
		},
		cli.BoolFlag{
			Name:        "no-key-cache",
			EnvVar:      "ZVELO_NO_KEY_CACHE",
			Usage:       "do not cache public keys when validating http signatures in callbacks",
			Destination: &c.callbackNoKeyCache,
		},
	}
}

func Command(appName string) cli.Command {
	c := cmd{appName: appName}

	return cli.Command{
		Name:   "receiver",
		Usage:  "listen for callbacks",
		Before: c.setup,
		Action: c.action,
		Flags:  c.Flags(),
	}
}

func (c *cmd) setup(cli *cli.Context) error {
	var keyCache callback.KeyCache

	if !c.callbackNoKeyCache {
		keyCache = callback.FileKeyCache(c.appName)
	}

	if !c.callbackNoValidate {
		c.keyGetter = callback.KeyGetter(keyCache)
	}

	return nil
}

func (c *cmd) action(_ *cli.Context) error {
	debugWriter := io.Writer(nil)

	if c.debug {
		debugWriter = os.Stderr
	}

	fmt.Fprintf(os.Stderr, "listening for callbacks at %s\n", c.listen) // #nosec

	return http.ListenAndServe(
		c.listen,
		callback.Middleware(c.keyGetter, c.callbackHandler(), debugWriter),
	)
}

func (c *cmd) callbackHandler() callback.Handler {
	return callback.HandlerFunc(func(w http.ResponseWriter, _ *http.Request, result *msg.QueryResult) {
		w.WriteHeader(http.StatusOK)

		if c.debug || zvelo.IsComplete(result) {
			results.Print(result, c.json)
		}
	})
}
