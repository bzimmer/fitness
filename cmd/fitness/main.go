package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
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

func newEngine(c *cli.Context) (*gin.Engine, error) {
	cfg, err := config(c)
	if err != nil {
		return nil, err
	}
	state, err := token(16)
	if err != nil {
		return nil, err
	}

	t, err := template.ParseFS(fitness.Content, "templates/index.html")
	if err != nil {
		return nil, err
	}

	store := cookie.NewStore([]byte(c.String("session-key")))
	config := &oauth2.Config{
		ClientID:     c.String("client-id"),
		ClientSecret: c.String("client-secret"),
		Scopes:       []string{"read_all,profile:read_all,activity:read_all"},
		RedirectURL:  fmt.Sprintf("%s/auth/callback", c.String("base-url")),
		Endpoint:     strava.Endpoint}

	engine := gin.Default()
	engine.Use(sessions.Sessions("default", store))
	engine.SetHTMLTemplate(t)
	engine.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("token") == nil {
			c.Redirect(http.StatusTemporaryRedirect, "/auth/login")
			return
		}
		c.HTML(http.StatusOK, "index.html", nil)
	})
	engine.GET("/auth/login", fitness.LoginHandler(config, state))
	engine.GET("/auth/logout", fitness.LogoutHandler(config, state))
	engine.GET("/auth/callback", fitness.AuthCallbackHandler(config, state))
	engine.GET("/scoreboard", fitness.ScoreboardHandler(config.ClientID, config.ClientSecret, cfg))
	return engine, nil
}

var serveCommand = &cli.Command{
	Name:  "serve",
	Usage: "Serve via http",
	Action: func(c *cli.Context) error {
		engine, err := newEngine(c)
		if err != nil {
			return err
		}
		u, err := url.Parse(c.String("base-url"))
		if err != nil {
			return err
		}
		_, port, _ := net.SplitHostPort(u.Host)
		address := fmt.Sprintf("0.0.0.0:%s", port)
		log.Info().Str("address", address).Msg("serving")
		return http.ListenAndServe(address, engine)
	},
}

var lambdaCommand = &cli.Command{
	Name:  "lambda",
	Usage: "Serve via lambda functions",
	Action: func(c *cli.Context) error {
		engine, err := newEngine(c)
		if err != nil {
			return err
		}
		gl := ginadapter.New(engine)
		lambda.Start(fitness.LambdaHandler(gl))
		return nil
	},
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
				Value:   "http://localhost:9001",
				Usage:   "Base URL",
				EnvVars: []string{"BASE_URL"},
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
			// @fix(bzimmer)
			if os.Getenv("NETLIFY") == "" {
				return serveCommand.Action(c)
			}
			return lambdaCommand.Action(c)
		},
		Commands: []*cli.Command{
			lambdaCommand,
			serveCommand,
		},
	}
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
