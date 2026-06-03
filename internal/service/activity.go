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
	"io"
	"net/http"
	"os/exec"
)

type ActivityService struct {
	ActivityRepo repo.IActivity
	log          *jsonlog.Logger
}

func NewActivityService(activityRepo repo.IActivity, log *jsonlog.Logger) ActivityService {
	return ActivityService{ActivityRepo: activityRepo, log: log}
}

func (s *ActivityService) Capture(ctx context.Context) error {
	s.log.Info("Capturing activity")
	activity, err := s.ActivityRepo.GetBacklitMetrics(ctx)
	if err != nil {
		return err
	}

	host := host.New(exec.Command)

	apiInput := input.Screentime{
		UserID:    "ID",
		Username:  host.Username,
		Date:      activity.Date.Format("2006-01-02"),
		MinutesOn: activity.MinutesOn,
	}

	jsonData, err := json.Marshal(apiInput)
	if err != nil {
		s.log.Error(err.Error())
		return err
	}

	s.log.Info(string(jsonData))

	request, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/screentime", bytes.NewBuffer(jsonData))
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
		bodyBytes, _ := io.ReadAll(resp.Body)
		s.log.Error(fmt.Sprintf(
			"API call failed with status %d and body %s",
			resp.StatusCode,
			string(bodyBytes),
		))
		return fmt.Errorf("API call failed with status %d and body %s ", resp.StatusCode, resp.Body)
	}
}
