package main

import (
	"github.com/johnmantios/micromanager/internal/config"
	"github.com/johnmantios/micromanager/internal/db"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/server"
	"github.com/johnmantios/micromanager/internal/service"
	"github.com/spf13/cobra"
)

func apiCmd(cfg *config.APIService, log *jsonlog.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "Serves",
		Long:  "Serves",
		Run: func(cmd *cobra.Command, args []string) {
			dbConnection, err := db.OpenDB(cfg.DB)
			if err != nil {
				log.Fatal(err, nil)
			}

			screentimeRepo, err := repo.NewScreentimeRepo(dbConnection)
			if err != nil {
				log.Fatal(err, nil)
			}

			screentimeService := service.NewScreentimeService(screentimeRepo, log)
			middleware := server.NewMiddleware(log)

			err = server.Serve(cfg.Port, log, screentimeService, middleware)
			if err != nil {
				return
			}
		},
	}
}
