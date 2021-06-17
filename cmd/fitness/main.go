package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/bzimmer/fitness"
	"github.com/bzimmer/gravl/pkg/providers/activity/strava"
)

type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func initLogging(c *cli.Context) error {
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
}

func credentials(c *cli.Context) (*Credentials, error) {
	fp, err := os.Open(c.String("config"))
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	val, _ := ioutil.ReadAll(fp)

	var creds Credentials
	err = json.Unmarshal(val, &creds)
	if err != nil {
		return nil, err
	}
	return &creds, nil
}

func main() {
	app := &cli.App{
		Name:     "fitness",
		HelpName: "fitness",
		Usage:    "Fitness Challenge",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "file with strava credentials",
			}},
		Before: initLogging,
		ExitErrHandler: func(c *cli.Context, err error) {
			if err == nil {
				return
			}
			log.Error().Err(err).Msg(c.App.Name)
		},
		Action: func(c *cli.Context) error {
			creds, err := credentials(c)
			if err != nil {
				return err
			}
			client, err := strava.NewClient(
				strava.WithTokenCredentials(creds.AccessToken, creds.RefreshToken, time.Now().Add(-1*time.Minute)),
				strava.WithClientCredentials(creds.ClientID, creds.ClientSecret),
				strava.WithAutoRefresh(c.Context))
			if err != nil {
				return err
			}
			board, err := fitness.Scoreboard(c.Context, client)
			if err != nil {
				return err
			}
			b, err := json.MarshalIndent(board, "", " ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(c.App.Writer, "%s\n", b)
			return err
		},
	}
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
