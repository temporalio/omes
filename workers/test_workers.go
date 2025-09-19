package workers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/loadgen"
	"go.uber.org/zap"
)

type workerPool struct {
	env *TestEnvironment

	mutex        sync.RWMutex
	buildOnce    map[cmdoptions.Language]*sync.Once
	buildErrs    map[cmdoptions.Language]error
	cleanupFuncs []func()
}

func NewWorkerPool(env *TestEnvironment) *workerPool {
	return &workerPool{
		env:       env,
		buildOnce: make(map[cmdoptions.Language]*sync.Once),
		buildErrs: make(map[cmdoptions.Language]error),
	}
}

func (w *workerPool) cleanup() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	for _, cleanupFunc := range w.cleanupFuncs {
		cleanupFunc()
	}
}

func (w *workerPool) ensureWorkerBuilt(
	t *testing.T,
	logger *zap.SugaredLogger,
	sdk cmdoptions.Language,
) error {
	w.mutex.Lock()
	once, exists := w.buildOnce[sdk]
	if !exists {
		once = new(sync.Once)
		w.buildOnce[sdk] = once
	}
	w.mutex.Unlock()

	once.Do(func() {
		baseDir := BaseDir(w.env.repoDir, sdk)
		buildDir := filepath.Join(baseDir, w.env.buildDirName())

		w.mutex.Lock()
		w.cleanupFuncs = append(w.cleanupFuncs, func() {
			if err := os.RemoveAll(buildDir); err != nil {
				fmt.Printf("Failed to clean up build dir for %s at %s: %v\n", sdk, buildDir, err)
			}
		})
		w.mutex.Unlock()

		builder := Builder{
			DirName:    w.env.buildDirName(),
			SdkOptions: cmdoptions.SdkOptions{Language: sdk},
			Logger:     logger.Named(fmt.Sprintf("%s-builder", sdk)),
		}

		buildCtx, buildCancel := context.WithTimeout(t.Context(), workerBuildTimeout)
		defer buildCancel()

		_, err := builder.Build(buildCtx, baseDir)

		w.mutex.Lock()
		w.buildErrs[sdk] = err
		w.mutex.Unlock()
	})

	w.mutex.RLock()
	err := w.buildErrs[sdk]
	w.mutex.RUnlock()

	return err
}

func (w *workerPool) startWorker(
	ctx context.Context,
	logger *zap.SugaredLogger,
	sdk cmdoptions.Language,
	taskQueueName string,
	scenarioInfo loadgen.ScenarioInfo,
) <-chan error {
	workerDone := make(chan error, 1)

	go func() {
		defer close(workerDone)
		baseDir := BaseDir(w.env.repoDir, sdk)
		runner := &Runner{
			Builder: Builder{
				DirName:    w.env.buildDirName(),
				SdkOptions: cmdoptions.SdkOptions{Language: sdk},
				Logger:     logger.Named(fmt.Sprintf("%s-worker-builder", sdk)),
			},
			TaskQueueName:            taskQueueName,
			GracefulShutdownDuration: workerShutdownTimeout,
			ScenarioID: cmdoptions.ScenarioID{
				Scenario: scenarioInfo.ScenarioName,
				RunID:    scenarioInfo.RunID,
			},
			ClientOptions: cmdoptions.ClientOptions{
				Address:   w.env.DevServerAddress(),
				Namespace: testNamespace,
			},
			LoggingOptions: cmdoptions.LoggingOptions{
				PreparedLogger: logger.Named(fmt.Sprintf("%s-worker", sdk)),
			},
		}
		workerDone <- runner.Run(ctx, baseDir)
	}()

	return workerDone
}
