package worker

import (
	"fmt"
	"io"
	"strings"
	"time"

	gocontext "golang.org/x/net/context"

	"github.com/mitchellh/multistep"
	"github.com/travis-ci/worker/backend"
	"github.com/travis-ci/worker/context"
)

type stepOpenLogWriter struct {
	logTimeout   time.Duration
	maxLogLength int
}

func (s *stepOpenLogWriter) Run(state multistep.StateBag) multistep.StepAction {
	ctx := state.Get("ctx").(gocontext.Context)
	buildJob := state.Get("buildJob").(Job)

	logWriter, err := buildJob.LogWriter(ctx)
	if err != nil {
		context.LoggerFromContext(ctx).WithField("err", err).Error("couldn't open a log writer")
		err := buildJob.Requeue()
		if err != nil {
			context.LoggerFromContext(ctx).WithField("err", err).Error("couldn't requeue job")
		}
		return multistep.ActionHalt
	}

	logWriter.SetTimeout(s.logTimeout)
	logWriter.SetMaxLogLength(s.maxLogLength)

	s.writeUsingWorker(state, logWriter)

	state.Put("logWriter", logWriter)

	return multistep.ActionContinue
}

func (s *stepOpenLogWriter) writeUsingWorker(state multistep.StateBag, w io.Writer) {
	instance := state.Get("instance").(backend.Instance)

	if hostname, ok := state.Get("hostname").(string); ok && hostname != "" {
		_, _ = writeFold(w, "worker_info", []byte(strings.Join([]string{
			"\033[33;1mWorker information\033[0m",
			fmt.Sprintf("hostname: %s", hostname),
			fmt.Sprintf("version: %s %s", VersionString, RevisionURLString),
			fmt.Sprintf("instance: %s", instance.ID()),
			fmt.Sprintf("startup: %v", instance.StartupDuration()),
		}, "\n")))
	}
}

func (s *stepOpenLogWriter) Cleanup(state multistep.StateBag) {
	logWriter, ok := state.Get("logWriter").(LogWriter)
	if ok {
		logWriter.Close()
	}
}
