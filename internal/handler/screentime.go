package handler

import (
	"context"
	"fmt"
	"github.com/johnmantios/micromanager/internal/server/input"
	"github.com/johnmantios/micromanager/internal/server/output"
)

func CreateScreentime(ctx context.Context, input *input.Screentime) (*output.Screentime, error) {
	return &output.Screentime{Message: fmt.Sprintf("Hello, %s!", input.Username)}, nil
}
