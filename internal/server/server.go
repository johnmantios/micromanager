package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/johnmantios/micromanager/internal/service"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type APIRouter struct {
	screentimeService       service.ScreentimeService
	applicationUsageService service.ApplicationUsageService
	middleware              middleware
}

func newAPIRouter(screentimeService service.ScreentimeService, applicationUsageService service.ApplicationUsageService, middleware2 middleware) *APIRouter {
	return &APIRouter{screentimeService: screentimeService, applicationUsageService: applicationUsageService, middleware: middleware2}
}

// NewHandler builds the fully wired HTTP handler for the API (routes plus
// middleware). It is useful for testing the API end to end and for embedding
// the API in a custom HTTP server.
func NewHandler(screentimeService service.ScreentimeService, applicationUsageService service.ApplicationUsageService, log *jsonlog.Logger) http.Handler {
	return newAPIRouter(screentimeService, applicationUsageService, NewMiddleware(log)).Routes()
}

func Serve(port int, log *jsonlog.Logger, screentimeService service.ScreentimeService, applicationUsageService service.ApplicationUsageService, middleware2 middleware) error {
	apiRouter := newAPIRouter(screentimeService, applicationUsageService, middleware2)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      apiRouter.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	wg := sync.WaitGroup{}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		log.Print("caught signal ", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		log.Print("completing background tasks ", srv.Addr)

		wg.Wait()
		shutdownError <- nil
	}()

	log.Info("Starting server %s", srv.Addr)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	log.Info("Stopped server %s", srv.Addr)

	return nil
}
