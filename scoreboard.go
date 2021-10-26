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

	var scores []*Activity
	detailsc := make(chan *Activity)
	activitiesc := make(chan *strava.Activity)

	actgrp, actctx := errgroup.WithContext(ctx)
	for i := 0; i < concurrency; i++ {
		actgrp.Go(func() error {
			for act := range activitiesc {
				week := b.week(act)
				if week == 0 {
					continue
				}
				log.Info().Str("name", act.Name).Int64("id", act.ID).Msg("query")
				details, err := client.Activity.Activity(actctx, act.ID)
				if err != nil {
					return err
				}
				detailsc <- &Activity{
					ID:       details.ID,
					Type:     details.Type,
					Name:     details.Name,
					Week:     week,
					Score:    b.score(details),
					Calories: b.calories(details),
				}
			}
			return nil
		})
	}
	actgrp.Go(func() error {
		defer close(activitiesc)
		start, end := b.config.DateRange()
		log.Info().Time("start", start).Time("end", end).Msg("date range")
		// yes, this order is correct
		opt := strava.WithDateRange(end.Add(time.Hour*24), start.Add(time.Hour*-24))
		acts := client.Activity.Activities(actctx, activity.Pagination{Total: n}, opt)
		for {
			select {
			case <-actctx.Done():
				return actctx.Err()
			case res, ok := <-acts:
				if !ok {
					return nil
				}
				if res.Err != nil {
					return res.Err
				}
				activitiesc <- res.Activity
			}
		}
	})

	detgrp, _ := errgroup.WithContext(ctx)
	detgrp.Go(func() error {
		defer close(detailsc)
		return actgrp.Wait()
	})
	detgrp.Go(func() error {
		for act := range detailsc {
			scores = append(scores, act)
		}
		return nil
	})
	if err := detgrp.Wait(); err != nil {
		return nil, err
	}
	return b.scoreboard(scores), nil
}
