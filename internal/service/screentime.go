package service

import (
	"context"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/model"
	"github.com/johnmantios/micromanager/internal/repo"
	"github.com/johnmantios/micromanager/internal/server/input"
	"github.com/johnmantios/micromanager/internal/server/output"
)

type ScreentimeService struct {
	ScreentimeRepo repo.IScreentime
	log            *jsonlog.Logger
}

func NewScreentimeService(screentimeRepo repo.IScreentime, log *jsonlog.Logger) ScreentimeService {
	return ScreentimeService{ScreentimeRepo: screentimeRepo, log: log}
}

func (s ScreentimeService) CreateScreentimeHandler(ctx context.Context, input *input.Screentime) (*output.Screentime, error) {
	screentimeModel := model.Screentime{
		UserID:    input.UserID,
		Username:  input.Username,
		Date:      input.Date,
		MinutesOn: input.Minutes,
	}

	return s.capture(ctx, &screentimeModel)
}

func (s ScreentimeService) capture(ctx context.Context, screentime *model.Screentime) (*output.Screentime, error) {
	err := s.ScreentimeRepo.InsertBacklitMetrics(ctx, screentime)
	if err != nil {
		return nil, err
	}

	apiOutput := output.Screentime{
		Message: "ok",
	}

	return &apiOutput, nil
}
