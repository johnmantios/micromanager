package main

import (
	"context"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/johnmantios/micromanager/internal/config"
	"github.com/johnmantios/micromanager/internal/db"
	"github.com/johnmantios/micromanager/internal/interrupt"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/service"
	"github.com/spf13/cobra"
	"time"
)

func pollCmd(cfg *config.ActivityDB, log *jsonlog.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "poll",
		Short: "Polls the local data source of the device",
		Long:  "Polls the local data source of the device for things like active screen time per application etc...",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())

			s, err := gocron.NewScheduler(
				gocron.WithLogger(log),
			)
			if err != nil {
				log.Error(err.Error())
			}

			dbConnection, err := db.OpenSQLiteDB(*cfg)
			if err != nil {
				log.Fatal(err, nil)
			}

			activityRepo, err := repo.NewActivityRepo(dbConnection)
			if err != nil {
				log.Fatal(err, nil)
			}

			activityService := service.NewActivityService(activityRepo, log)

			job, err := s.NewJob(
				gocron.CronJob(cfg.PullPeriodCronSchedule, false),
				gocron.NewTask(activityService.Capture, ctx),
				gocron.WithEventListeners(
					gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
						log.Error(err.Error(), "jobID", jobID, "jobName", jobName)
					}),
				),
			)
			if err != nil {
				log.Fatal(err, "cron", cfg.PullPeriodCronSchedule)
			}
			_ = job

			s.Start()

			select {
			case <-time.After(time.Minute):
			}

			<-interrupt.WaitForInterrupt(cancel, log)
		},
	}
}
