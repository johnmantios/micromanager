package main

import (
	"github.com/johnmantios/micromanager/internal/config"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/server"
	"github.com/spf13/cobra"
)

func apiCmd(cfg *config.APIService, log *jsonlog.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "Serves",
		Long:  "Serves",
		Run: func(cmd *cobra.Command, args []string) {

			err := server.Serve(cfg.Port, log)
			if err != nil {
				return
			}
		},
	}
}
