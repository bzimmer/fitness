package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	echoadapter "github.com/awslabs/aws-lambda-go-api-proxy/echo"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"

	"github.com/bzimmer/fitness"
	"github.com/bzimmer/gravl/pkg/providers/activity/strava"
)

func config(c *cli.Context) (*fitness.Config, error) {
	var err error
	var val []byte
	switch c.IsSet("config") {
	case true:
		log.Info().Str("file", c.String("config")).Msg("config")
		var fp *os.File
		fp, err = os.Open(c.String("config"))
		if err != nil {
			return nil, err
		}
		defer fp.Close()
		val, _ = io.ReadAll(fp)
	case false:
		log.Info().Str("file", "etc/scoreboard.json").Msg("config")
		val, err = fitness.Content.ReadFile("etc/scoreboard.json")
		if err != nil {
			return nil, err
		}
	}
	var cfg fitness.Config
	err = json.Unmarshal(val, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// token produces a random token of length `n`
func token(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func newEngine(c *cli.Context) (*echo.Echo, error) {
	cfg, err := config(c)
	if err != nil {
		return nil, err
	}
	state, err := token(16)
	if err != nil {
		return nil, err
	}

	baseURL := c.String("base-url")
	log.Info().Str("baseURL", baseURL).Msg("found baseURL")
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	log.Info().Str("path", u.Path).Msg("root path")

	config := &oauth2.Config{
		ClientID:     c.String("client-id"),
		ClientSecret: c.String("client-secret"),
		Scopes:       []string{"read_all,profile:read_all,activity:read_all"},
		RedirectURL:  baseURL + "/callback",
		Endpoint:     strava.Endpoint}

	engine := echo.New()
	engine.Pre(middleware.RemoveTrailingSlash())
	engine.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "time=${time_rfc3339}, method=${method}, uri=${uri}, path=${path}, status=${status}\n",
	}))
	engine.HTTPErrorHandler = func(err error, c echo.Context) {
		engine.DefaultHTTPErrorHandler(err, c)
		log.Error().Err(err).Msg("error")
	}

	store := sessions.NewCookieStore([]byte(c.String("session-key")))
	store.Options.HttpOnly = true
	store.Options.Secure = true
	engine.Use(session.Middleware(store))

	base := engine.Group(u.Path)
	base.GET("/login", fitness.LoginHandler(config, state))
	base.GET("/logout", fitness.LogoutHandler(config, state, "/"))
	base.GET("/callback", fitness.AuthCallbackHandler(config, state, "/"))
	base.GET("/scoreboard", fitness.ScoreboardHandler(config.ClientID, config.ClientSecret, cfg))

	return engine, nil
}

func serve(c *cli.Context) error {
	engine, err := newEngine(c)
	if err != nil {
		return err
	}
	engine.Static("/", "public")
	address := fmt.Sprintf(":%d", c.Int("port"))
	log.Info().Str("address", address).Msg("http server")
	return http.ListenAndServe(address, engine)
}

func function(c *cli.Context) error {
	engine, err := newEngine(c)
	if err != nil {
		return err
	}
	log.Info().Msg("lambda function")
	gl := echoadapter.New(engine)
	lambda.Start(func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return gl.ProxyWithContext(ctx, req)
	})
	return nil
}

func main() {
	app := &cli.App{
		Name:     "fitness",
		HelpName: "fitness",
		Usage:    "Fitness Challenge",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "client-id",
				Required: true,
				Usage:    "client id",
				EnvVars:  []string{"STRAVA_CLIENT_ID"},
			},
			&cli.StringFlag{
				Name:     "client-secret",
				Required: true,
				Usage:    "client secret",
				EnvVars:  []string{"STRAVA_CLIENT_SECRET"},
			},
			&cli.StringFlag{
				Name:     "session-key",
				Required: true,
				Usage:    "session keypair",
				EnvVars:  []string{"FITNESS_SESSION_KEY"},
			},
			&cli.StringFlag{
				Name:    "base-url",
				Value:   "http://localhost",
				Usage:   "Base URL",
				EnvVars: []string{"BASE_URL"},
			},
			&cli.IntFlag{
				Name:  "port",
				Value: 0,
				Usage: "port on which to run",
			},
			&cli.StringFlag{
				Name:  "config",
				Usage: "file with fitness configuration parameters",
			},
		},
		ExitErrHandler: func(c *cli.Context, err error) {
			if err == nil {
				return
			}
			log.Error().Err(err).Msg(c.App.Name)
		},
		Before: func(c *cli.Context) error {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			zerolog.DurationFieldUnit = time.Millisecond
			zerolog.DurationFieldInteger = false
			log.Logger = log.Output(
				zerolog.ConsoleWriter{
					Out:        c.App.ErrWriter,
					NoColor:    false,
					TimeFormat: time.RFC3339,
				},
			)
			return nil
		},
		Action: func(c *cli.Context) error {
			if c.IsSet("port") {
				return serve(c)
			}
			return function(c)
		},
	}
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
