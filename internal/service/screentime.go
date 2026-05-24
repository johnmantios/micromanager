package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/johnmantios/micromanager/internal/host"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/server/input"
	"net/http"
	"os/exec"
)

type ScreentimeService struct {
	ActivityRepo repo.IActivity
	log          *jsonlog.Logger
}

func NewScreentimeService(activityRepo repo.IActivity, log *jsonlog.Logger) ScreentimeService {
	return ScreentimeService{ActivityRepo: activityRepo, log: log}
}

func (s *ScreentimeService) Capture(ctx context.Context) error {
	s.log.Info("Capturing screentime")
	screentime, err := s.ActivityRepo.GetBacklitMetrics(ctx)
	if err != nil {
		return err
	}

	host := host.New(exec.Command)

	apiInput := input.Screentime{
		UserID:   "ID",
		Username: host.Username,
		Date:     screentime.Date,
		Minutes:  screentime.MinutesOn,
	}

	jsonData, err := json.Marshal(apiInput)
	if err != nil {
		s.log.Error(err.Error())
		return err
	}

	request, err := http.NewRequest(http.MethodPost, "some_URL", bytes.NewBuffer(jsonData))
	if err != nil {
		s.log.Error(err.Error())
		return err
	}

	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return nil
	} else {
		return fmt.Errorf("API call failed with status %d and body %s ", resp.StatusCode, resp.Body)
	}
}
