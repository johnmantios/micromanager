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

type ApplicationUsageService struct {
	ApplicationUsageRepo repo.IApplicationUsage
	log                  *jsonlog.Logger
}

func NewApplicationUsageService(applicationUsageRepo repo.IApplicationUsage, log *jsonlog.Logger) ApplicationUsageService {
	return ApplicationUsageService{ApplicationUsageRepo: applicationUsageRepo, log: log}
}

func (s ApplicationUsageService) CreateAppUsageHandler(ctx context.Context, input *input.AppUsage) (*output.AppUsage, error) {
	s.log.Print("input is: ", input)

	applications := make([]model.ApplicationUsage, 0, len(input.Body.Applications))
	for _, application := range input.Body.Applications {
		applications = append(applications, model.ApplicationUsage{
			BundleID:  application.BundleID,
			MinutesOn: application.MinutesOn,
		})
	}

	usage := model.AppUsage{
		UserID:       input.Body.UserID,
		Username:     input.Body.Username,
		Date:         input.Body.Date,
		Applications: applications,
	}

	if err := s.ApplicationUsageRepo.InsertApplicationUsage(ctx, &usage); err != nil {
		return nil, huma.Error500InternalServerError("Could not insert application usage in database", err)
	}

	apiOutput := output.AppUsage{}
	apiOutput.Body.Message = "ok"

	return &apiOutput, nil
}
