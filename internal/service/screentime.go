package service

import (
	"context"
	"github.com/danielgtaylor/huma/v2"
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
	s.log.Print("input is: ", input)

	screentimeModel := model.Screentime{
		UserID:    input.UserID,
		Username:  input.Username,
		Date:      input.Date,
		MinutesOn: input.MinutesOn,
	}

	return s.createScreenTimeHandler(ctx, &screentimeModel)
}

func (s ScreentimeService) createScreenTimeHandler(ctx context.Context, screentime *model.Screentime) (*output.Screentime, error) {
	err := s.ScreentimeRepo.InsertBacklitMetrics(ctx, screentime)
	if err != nil {
		return nil, huma.Error500InternalServerError("Could not insert backlit metrics in database", err)
	}

	apiOutput := output.Screentime{
		Message: "ok",
	}

	return &apiOutput, nil
}
