package tokensourcer

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/urfave/cli"

	"golang.org/x/oauth2"

	zapi "zvelo.io/go-zapi"
	"zvelo.io/go-zapi/clientauth"
	"zvelo.io/go-zapi/tokensource"
	"zvelo.io/go-zapi/userauth"
)

type TokenSourcer interface {
	Flags() []cli.Flag
	TokenSource() oauth2.TokenSource
	Verifier(context.Context) (*oidc.IDTokenVerifier, error)
}

func New(appName string, debug, insecureSkipVerify *bool, scope ...string) TokenSourcer {
	return &data{
		appName:            appName,
		debug:              debug,
		insecureSkipVerify: insecureSkipVerify,
		defaultScopes:      scope,
	}
}

type data struct {
	// cached data
	tokenSource oauth2.TokenSource
	verifier    *oidc.IDTokenVerifier

	// passed to constructor
	appName            string
	debug              *bool
	insecureSkipVerify *bool

	// from flags
	accessToken        string
	oauth2             oauth2.Config
	mockNoCredentials  bool
	useUserCredentials bool
	redirectURL        string
	callbackAddr       string
	noOpenBrowser      bool
	noCacheToken       bool
	scopesFlag         cli.StringSlice
	oidcIssuer         string

	defaultScopes []string
}

func (d *data) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "auth-url",
			EnvVar:      "ZVELO_AUTH_URL",
			Usage:       "oauth2 auth url",
			Value:       zapi.Endpoint.AuthURL,
			Destination: &d.oauth2.Endpoint.AuthURL,
		},
		cli.StringFlag{
			Name:        "token-url",
			EnvVar:      "ZVELO_TOKEN_URL",
			Usage:       "oauth2 token url",
			Value:       zapi.Endpoint.TokenURL,
			Destination: &d.oauth2.Endpoint.TokenURL,
		},
		cli.StringFlag{
			Name:        "oidc-issuer",
			EnvVar:      "ZVELO_OIDC_ISSUER",
			Usage:       "oidc issuer url",
			Value:       "https://auth.zvelo.com",
			Destination: &d.oidcIssuer,
		},
		cli.StringFlag{
			Name:        "client-id",
			EnvVar:      "ZVELO_CLIENT_ID",
			Usage:       "oauth2 client id",
			Destination: &d.oauth2.ClientID,
		},
		cli.StringFlag{
			Name:        "client-secret",
			EnvVar:      "ZVELO_CLIENT_SECRET",
			Usage:       "oauth2 client secret",
			Destination: &d.oauth2.ClientSecret,
		},
		cli.StringFlag{
			Name:        "access-token",
			EnvVar:      "ZVELO_ACCESS_TOKEN",
			Usage:       "explicitly provide an access token. this should rarely be used as it will override client-id, client-secret and user-credentials",
			Destination: &d.accessToken,
		},
		cli.BoolFlag{
			Name:        "mock-no-credentials",
			Usage:       "when querying against the mock server, which does not require credentials, do not attempt to get a token",
			Destination: &d.mockNoCredentials,
		},
		cli.BoolFlag{
			Name:        "use-user-credentials",
			EnvVar:      "ZVELO_USE_USER_CREDENTIALS",
			Usage:       "use user, 3 legged oauth2, credentials instead of client credentials",
			Destination: &d.useUserCredentials,
		},
		cli.StringFlag{
			Name:        "oauth2-callback-url",
			EnvVar:      "ZVELO_OAUTH2_CALLBACK_URL",
			Usage:       "url that server will redirect to for oauth2 callbacks",
			Value:       userauth.DefaultRedirectURL,
			Destination: &d.redirectURL,
		},
		cli.StringFlag{
			Name:        "oauth2-callback-addr",
			EnvVar:      "ZVELO_OAUTH2_CALLBACK_ADDR",
			Usage:       "addr:port that server will listen to for oauth2 callbacks",
			Value:       userauth.DefaultCallbackAddr,
			Destination: &d.callbackAddr,
		},
		cli.BoolFlag{
			Name:        "oauth2-no-open-in-browser",
			EnvVar:      "ZVELO_OAUTH2_NO_OPEN_IN_BROWSER",
			Usage:       "don't open the auth url in the browser",
			Destination: &d.noOpenBrowser,
		},
		cli.BoolFlag{
			Name:        "no-cache-token",
			EnvVar:      "ZVELO_NO_CACHE_TOKEN",
			Usage:       "don't cache received oauth2 tokens to the filesystem",
			Destination: &d.noCacheToken,
		},
		cli.StringSliceFlag{
			Name:   "scope",
			EnvVar: "ZVELO_SCOPES",
			Usage:  "scopes to request with the token, may be repeated (default: " + strings.Join(d.defaultScopes, ", ") + ")",
			Value:  &d.scopesFlag,
		},
	}
}

func (d *data) scopes() []string {
	var s []string

	if len(d.scopesFlag) > 0 {
		s = d.scopesFlag
	} else {
		s = d.defaultScopes
	}

	scopes := make([]string, len(s))
	copy(scopes, s)

	return scopes
}

func (d *data) TokenSource() oauth2.TokenSource {
	scopes := d.scopes()

	if d.tokenSource != nil || d.mockNoCredentials {
		return nil
	}

	var cacheName string

	if d.accessToken != "" {
		d.tokenSource = oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: d.accessToken,
		})
	} else if d.useUserCredentials {
		cacheName = "user"
		userOpts := []userauth.Option{
			userauth.WithRedirectURL(d.redirectURL),
			userauth.WithScope(scopes...),
			userauth.WithCallbackAddr(d.callbackAddr),
			userauth.WithEndpoint(d.oauth2.Endpoint),
		}

		if d.noOpenBrowser {
			userOpts = append(userOpts, userauth.WithoutOpen())
		}

		if *d.debug {
			userOpts = append(userOpts, userauth.WithDebug(os.Stderr))
		}

		d.tokenSource = userauth.TokenSource(context.Background(), d.oauth2.ClientID, d.oauth2.ClientSecret, userOpts...)
	} else {
		cacheName = "client"
		d.tokenSource = clientauth.ClientCredentials(
			context.Background(),
			d.oauth2.ClientID,
			d.oauth2.ClientSecret,
			clientauth.WithScope(scopes...),
			clientauth.WithTokenURL(d.oauth2.Endpoint.TokenURL),
		)
	}

	if d.tokenSource != nil {
		if d.accessToken == "" {
			if !d.noCacheToken {
				d.tokenSource = tokensource.FileCache(d.tokenSource, d.appName, cacheName, scopes...)
			}

			d.tokenSource = oauth2.ReuseTokenSource(nil, d.tokenSource)
		}

		if *d.debug {
			d.tokenSource = tokensource.Debug(os.Stderr, d.tokenSource)
		}
	}

	return d.tokenSource
}

func (d *data) Verifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	scopes := d.scopes()

	if d.verifier != nil {
		return d.verifier, nil
	}

	for _, s := range scopes {
		if s != "openid" {
			continue
		}

		if *d.insecureSkipVerify {
			if t, ok := http.DefaultTransport.(*http.Transport); ok {
				prev := t.TLSClientConfig
				t.TLSClientConfig = &tls.Config{
					InsecureSkipVerify: true,
				}

				defer func() {
					t.TLSClientConfig = prev
				}()
			}
		}

		provider, err := oidc.NewProvider(ctx, d.oidcIssuer)
		if err != nil {
			return nil, err
		}

		d.verifier = provider.Verifier(&oidc.Config{ClientID: d.oauth2.ClientID})
		break
	}

	return d.verifier, nil
}
