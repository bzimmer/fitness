package fitness

import (
	"context"
	"math"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/bzimmer/activity"
	"github.com/bzimmer/activity/strava"
)

const (
	n           = 100
	concurrency = 25
)

type Scoreboard struct {
	config *Config
}

func NewScoreboard(config *Config) *Scoreboard {
	return &Scoreboard{config: config}
}

func (b *Scoreboard) score(act *strava.Activity) int {
	var val float64
	movingTime := time.Minute * time.Duration(math.Ceil(act.MovingTime.Minutes()))
	switch act.Type {
	case "Hike", "Ride":
		val = 1.75
	case "Run":
		val = 1.50
	default:
		val = 1.0
	}
	for _, epic := range b.config.Epic {
		if epic.Type == act.Type {
			if movingTime > time.Minute*time.Duration(epic.Minutes) {
				val = epic.Multiplier
			}
		}
	}
	return int(movingTime.Minutes() * val)
}

func (b *Scoreboard) week(act *strava.Activity) int {
	for i, wk := range b.config.Weeks {
		if act.StartDate.After(wk.Start) && act.StartDate.Before((wk.End)) {
			return i + 1
		}
	}
	return 0
}

func (b *Scoreboard) calories(act *strava.Activity) int {
	for _, cal := range b.config.Calories {
		if cal.ID == act.ID {
			return cal.Override
		}
	}
	return int(act.Calories)
}

func (b *Scoreboard) scoreboard(acts []*Activity) []*Week {
	// group all activities into weeks
	w := make(map[int][]*Activity)
	for _, act := range acts {
		w[act.Week] = append(w[act.Week], act)
	}
	// summarize score and calories
	var res []*Week
	for wk, acts := range w {
		var cal, score int
		for _, act := range acts {
			score += act.Score
			cal += act.Calories
		}
		m := &Week{
			Week:       wk,
			Score:      score,
			Calories:   cal,
			Activities: acts,
		}
		res = append(res, m)
	}
	return res
}

// Scoreboard returns weekly scores and calories for the current athlete
func (b *Scoreboard) Scoreboard(c context.Context, client *strava.Client) ([]*Week, error) {
	ctx, cancel := context.WithTimeout(c, 2*time.Minute)
	defer cancel()

	detc := make(chan *Activity, n)
	sumc := make(chan *strava.Activity, n)

	grp, ctx := errgroup.WithContext(ctx)
	for i := 0; i < concurrency; i++ {
		grp.Go(func() error {
			for act := range sumc {
				week := b.week(act)
				if week == 0 {
					continue
				}
				log.Info().Str("name", act.Name).Int64("id", act.ID).Msg("query")
				act, err := client.Activity.Activity(ctx, act.ID)
				if err != nil {
					return err
				}
				detc <- &Activity{
					ID:       act.ID,
					Type:     act.Type,
					Name:     act.Name,
					Week:     week,
					Score:    b.score(act),
					Calories: b.calories(act),
				}
			}
			return nil
		})
	}

	err := func() error {
		defer close(sumc)
		start, end := b.config.DateRange()
		// yes this order is correct
		opt := strava.WithDateRange(end.Add(time.Hour*24), start.Add(time.Hour*-24))
		acts := client.Activity.Activities(ctx, activity.Pagination{Total: n}, opt)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res, ok := <-acts:
				if !ok {
					return nil
				}
				if res.Err != nil {
					return res.Err
				}
				sumc <- res.Activity
			}
		}
	}()

	if err != nil {
		return nil, err
	}

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	close(detc)

	var scores []*Activity
	for act := range detc {
		scores = append(scores, act)
	}
	return b.scoreboard(scores), nil
}
