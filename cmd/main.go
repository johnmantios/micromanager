package main

import (
	"fmt"
	"github.com/johnmantios/micromanager/internal/config"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	cfg, err := config.LoadEnv()
	if err != nil {
		panic(fmt.Sprintf("Loading config from environment failed: %s", err.Error()))
	}

	log := jsonlog.New(os.Stdout, cfg.LogLevel)

	rootCmd := &cobra.Command{Use: "micromanager"}

	rootCmd.AddCommand(pollCmd(cfg, log))
	rootCmd.AddCommand(apiCmd(&cfg.APIService, log))

	err = rootCmd.Execute()
	if err != nil {
		log.Fatal(err, map[string]string{
			"message": "Command failed",
		})
	}
}
