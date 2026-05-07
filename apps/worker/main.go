package main

import (
	"context"
	"fmt"
	"os"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/mortgage-application/apps/worker/internal/mortgage"
	"github.com/mrsimonemms/mortgage-application/apps/worker/internal/mortgage/activities"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	TaskQueue = "mortgage-application"

	// WorkerProfileV1 runs the original mortgage workflow without a property
	// valuation step. WorkerProfileV2 runs the same workflow with a property
	// valuation step inserted between the credit check and the offer
	// reservation. Both profiles register their workflow under the same name
	// (mortgage.WorkflowTypeName); Temporal Worker Deployment Versioning
	// routes pinned executions back to their originating worker version while
	// new executions go to the current deployment version.
	WorkerProfileV1 = "v1"
	WorkerProfileV2 = "v2"

	envWorkerProfile        = "WORKER_PROFILE"
	envWorkerDeploymentName = "WORKER_DEPLOYMENT_NAME"
	envWorkerBuildID        = "WORKER_BUILD_ID"
)

// workerProfile holds the profile selection and the Worker Deployment Version
// identifiers for this worker process. Values are loaded once at startup; the
// worker fails fast if any are missing or invalid so the operator notices a
// misconfiguration immediately rather than silently running an unversioned
// worker.
type workerProfile struct {
	Profile        string
	DeploymentName string
	BuildID        string
}

func loadWorkerProfile() (workerProfile, error) {
	p := workerProfile{
		Profile:        os.Getenv(envWorkerProfile),
		DeploymentName: os.Getenv(envWorkerDeploymentName),
		BuildID:        os.Getenv(envWorkerBuildID),
	}

	if p.Profile == "" {
		return p, fmt.Errorf("%s is required (expected %q or %q)", envWorkerProfile, WorkerProfileV1, WorkerProfileV2)
	}
	if p.Profile != WorkerProfileV1 && p.Profile != WorkerProfileV2 {
		return p, fmt.Errorf("%s=%q is invalid (expected %q or %q)", envWorkerProfile, p.Profile, WorkerProfileV1, WorkerProfileV2)
	}
	if p.DeploymentName == "" {
		return p, fmt.Errorf("%s is required for Worker Deployment Versioning", envWorkerDeploymentName)
	}
	if p.BuildID == "" {
		return p, fmt.Errorf("%s is required for Worker Deployment Versioning", envWorkerBuildID)
	}

	return p, nil
}

// registerWorkflowForProfile registers the correct workflow implementation for
// the active profile under the shared workflow type name. Both versions are
// pinned by default: existing executions stay on the worker version that
// started them, and new executions are routed to the current Worker
// Deployment Version. This is what gives the demo "v1 keeps running, v2
// handles new applications" behaviour without any patch-style versioning.
func registerWorkflowForProfile(w worker.Worker, profile string) error {
	opts := workflow.RegisterOptions{
		Name:               mortgage.WorkflowTypeName,
		VersioningBehavior: workflow.VersioningBehaviorPinned,
	}

	switch profile {
	case WorkerProfileV1:
		w.RegisterWorkflowWithOptions(mortgage.MortgageApplicationWorkflow, opts)
	case WorkerProfileV2:
		w.RegisterWorkflowWithOptions(mortgage.MortgageApplicationWorkflowV2, opts)
	default:
		return fmt.Errorf("unsupported worker profile %q", profile)
	}

	return nil
}

func exec() error {
	logLevel := "info"
	if l, ok := os.LookupEnv("LOG_LEVEL"); ok {
		logLevel = l
	}
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)

	wp, err := loadWorkerProfile()
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Worker versioning configuration is invalid",
		}
	}

	log.Info().
		Str("profile", wp.Profile).
		Str("deploymentName", wp.DeploymentName).
		Str("buildId", wp.BuildID).
		Msg("Worker Deployment Versioning configured")

	opts := []temporal.Options{
		temporal.WithZerolog(&log.Logger),
		temporal.WithPrometheusMetrics("0.0.0.0:9090", "", nil),
	}

	if keysPath, ok := os.LookupEnv("KEYS_PATH"); ok {
		keys, err := aes.ReadKeyFile(keysPath)
		if err != nil {
			return gh.FatalError{
				Cause: err,
				Msg:   "Unable to read keys file",
			}
		}
		opts = append(opts, temporal.WithDataAndFailureConverter(aes.DataConverter(keys)))
	}

	// The client is a heavyweight object that should be created once per process.
	c, err := temporal.NewConnectionWithEnvvars(opts...)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to create client",
		}
	}
	defer c.Close()

	w := worker.New(c, TaskQueue, worker.Options{
		DeploymentOptions: worker.DeploymentOptions{
			UseVersioning: true,
			Version: worker.WorkerDeploymentVersion{
				DeploymentName: wp.DeploymentName,
				BuildID:        wp.BuildID,
			},
			DefaultVersioningBehavior: workflow.VersioningBehaviorPinned,
		},
	})

	if err := registerWorkflowForProfile(w, wp.Profile); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to register workflow for profile",
		}
	}

	acts, err := activities.NewActivities(wp.Profile)
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Unable to construct activities",
		}
	}

	w.RegisterActivity(acts)

	// Start the healthcheck server in a separate goroutine
	temporal.NewHealthCheck(context.Background(), []string{TaskQueue}, "0.0.0.0:9000", c)

	log.Info().
		Str("taskQueue", TaskQueue).
		Str("profile", wp.Profile).
		Str("deploymentName", wp.DeploymentName).
		Str("buildId", wp.BuildID).
		Msg("Worker listening on task queue")
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
