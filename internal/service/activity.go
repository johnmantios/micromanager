package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"

	"github.com/johnmantios/micromanager/internal/host"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/server/input"
)

const (
	defaultAPIBaseURL  = "http://localhost:8080"
	defaultHTTPTimeout = 10 * time.Second
)

type ActivityService struct {
	ActivityRepo repo.IActivity
	log          *jsonlog.Logger
	apiBaseURL   string
	httpClient   *http.Client
}

// NewActivityService builds an ActivityService that talks to the default API address.
func NewActivityService(activityRepo repo.IActivity, log *jsonlog.Logger) ActivityService {
	return NewActivityServiceWithAPI(activityRepo, log, defaultAPIBaseURL, nil)
}

// NewActivityServiceWithAPI builds an ActivityService against an explicit API address and HTTP client.
func NewActivityServiceWithAPI(activityRepo repo.IActivity, log *jsonlog.Logger, apiBaseURL string, client *http.Client) ActivityService {
	if apiBaseURL == "" {
		apiBaseURL = defaultAPIBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return ActivityService{
		ActivityRepo: activityRepo,
		log:          log,
		apiBaseURL:   apiBaseURL,
		httpClient:   client,
	}
}

// CaptureTotalScreentime reads the daily backlit metric and sends it to the screentime endpoint.
func (s *ActivityService) CaptureTotalScreentime(ctx context.Context) error {
	s.log.Info("Capturing activity")
	activity, err := s.ActivityRepo.GetBacklitMetrics(ctx)
	if err != nil {
		return err
	}

	host := host.New(exec.Command)

	apiInput := input.Screentime{}
	apiInput.Body.UserID = host.Username
	apiInput.Body.Username = host.Username
	apiInput.Body.Date = activity.Date.Format("2006-01-02")
	apiInput.Body.MinutesOn = activity.MinutesOn

	return s.postJSON(ctx, "/v1/screentime", apiInput.Body)
}

// CaptureApplicationUsage reads the daily per application usage and sends it to the app usage endpoint.
func (s *ActivityService) CaptureApplicationUsage(ctx context.Context) error {
	s.log.Info("Capturing application usage")

	applications, err := s.ActivityRepo.GetMinutesPerApplication(ctx)
	if err != nil {
		return err
	}

	if len(applications) == 0 {
		s.log.Info("No application usage to capture")
		return nil
	}

	apiInput := input.AppUsage{}
	host := host.New(exec.Command)
	apiInput.Body.UserID = host.Username
	apiInput.Body.Username = host.Username
	apiInput.Body.Date = time.Now().Format("2006-01-02")
	apiInput.Body.Applications = make([]input.AppUsageItem, 0, len(applications))
	for _, application := range applications {
		apiInput.Body.Applications = append(apiInput.Body.Applications, input.AppUsageItem{
			BundleID:  application.BundleID,
			MinutesOn: application.MinutesOn,
		})
	}

	s.log.Debug(fmt.Sprintf("Application usage: %+v", apiInput.Body))

	return s.postJSON(ctx, "/v1/app-usage", apiInput.Body)
}

// CaptureAll runs every collector and joins their errors.
func (s *ActivityService) CaptureAll(ctx context.Context) error {
	return errors.Join(s.CaptureTotalScreentime(ctx), s.CaptureApplicationUsage(ctx))
}

func (s *ActivityService) postJSON(ctx context.Context, path string, payload any) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		s.log.Error(err.Error())
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiBaseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		s.log.Error(err.Error())
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	s.log.Error(fmt.Sprintf("API call failed with status %d and body %s", resp.StatusCode, string(bodyBytes)))
	return fmt.Errorf("API call failed with status %d and body %s", resp.StatusCode, string(bodyBytes))
}
