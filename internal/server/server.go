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
	screentimeService service.ScreentimeService
}

func newAPIRouter(screentimeService service.ScreentimeService) *APIRouter {
	return &APIRouter{screentimeService: screentimeService}
}

func Serve(port int, log *jsonlog.Logger, screentimeService service.ScreentimeService) error {
	apiRouter := newAPIRouter(screentimeService)

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

		log.Print("caught signal %s", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		log.Print("completing background tasks %s", srv.Addr)

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
