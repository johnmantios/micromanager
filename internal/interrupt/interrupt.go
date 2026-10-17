package interrupt

import (
	"context"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"os"
	"os/signal"
	"syscall"
)

func WaitForInterrupt(cancelFn context.CancelFunc, log *jsonlog.Logger) chan os.Signal {
	signalCh := make(chan os.Signal, 10)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	defer func() {
		signal.Stop(signalCh)
		close(signalCh)
	}()

	s := <-signalCh
	log.Info("received termination signal: %s", s)

	cancelFn()
	return signalCh
}
