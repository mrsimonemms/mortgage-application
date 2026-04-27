package main

import (
	"context"
	"os"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/worker"
)

const (
	TaskQueue = "hello-world"
)

func exec() error {
	// The client is a heavyweight object that should be created once per process.
	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithZerolog(&log.Logger),
		temporal.WithPrometheusMetrics("0.0.0.0:9090", "", nil),
	)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to create client",
		}
	}
	defer c.Close()

	w := worker.New(c, TaskQueue, worker.Options{})

	// Start the healthcheck server in a separate goroutine
	temporal.NewHealthCheck(context.Background(), []string{TaskQueue}, "0.0.0.0:9000", c)

	log.Info().Str("taskQueue", TaskQueue).Msg("Worker listening on task queue")
	if err := w.Run(worker.InterruptCh()); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Worker stopped",
		}
	}

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}
