package scheduler

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/viniciushammett/k8s-pod-restarter/internal/config"
	"github.com/viniciushammett/k8s-pod-restarter/internal/logger"
	"github.com/viniciushammett/k8s-pod-restarter/internal/restarter"
)

func Run(ctx context.Context, log *logger.Logger, cfg *config.Config, r *restarter.Restarter) error {
	c := cron.New(cron.WithParser(cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)))
	for _, job := range cfg.Jobs {
		j := job // capture
		_, err := c.AddFunc(j.Schedule, func() {
			log.Info().Str("job", j.Name).Msg("running scheduled restart")
			opts, err := toOptions(j)
			if err != nil {
				log.Error().Err(err).Str("job", j.Name).Msg("invalid job options")
				return
			}
			if err := r.RestartPods(ctx, opts); err != nil {
				log.Error().Err(err).Str("job", j.Name).Msg("restart failed")
			}
		})
		if err != nil {
			log.Error().Err(err).Str("job", j.Name).Msg("cron add failed")
		}
	}
	c.Start()
	<-ctx.Done()
	ctxStop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.Stop().Done() // graceful stop
	<-ctxStop.Done()
	return nil
}

func toOptions(j config.Job) (restarter.Options, error) {
	var maxAge, grace time.Duration
	var err error
	if j.MaxAge != "" {
		maxAge, err = time.ParseDuration(j.MaxAge)
		if err != nil {
			return restarter.Options{}, err
		}
	}
	if j.GracePeriod != "" {
		grace, err = time.ParseDuration(j.GracePeriod)
		if err != nil {
			return restarter.Options{}, err
		}
	} else {
		grace = 30 * time.Second
	}
	return restarter.Options{
		Namespace:   j.Namespace,
		Selector:    j.Selector,
		DryRun:      j.DryRun,
		Force:       j.Force,
		MaxAge:      maxAge,
		GracePeriod: grace,
	}, nil
}