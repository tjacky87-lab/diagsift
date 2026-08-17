package collector

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/tjacky87-lab/diagsift/internal/manifest"
	"github.com/tjacky87-lab/diagsift/internal/policy"
	"github.com/tjacky87-lab/diagsift/internal/report"
)

type boundedBuffer struct {
	buffer    bytes.Buffer
	remaining int64
	truncated bool
}

func newBoundedBuffer(limit int64) *boundedBuffer {
	return &boundedBuffer{remaining: limit}
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	if int64(len(data)) > b.remaining {
		data = data[:max(b.remaining, 0)]
		b.truncated = true
	}
	if len(data) > 0 {
		_, _ = b.buffer.Write(data)
		b.remaining -= int64(len(data))
	}
	return original, nil
}

func collectCommand(ctx context.Context, sink *Sink, spec manifest.Collector, sanitizer Sanitizer) {
	if err := policy.ValidateCommand(spec.Executable, spec.Args); err != nil {
		sink.Error(spec.ID, "command-policy", "command failed execution policy")
		return
	}
	commandContext, cancel := context.WithTimeout(ctx, spec.Timeout.Duration)
	defer cancel()

	command := exec.Command(spec.Executable, spec.Args...)
	command.Env = MinimalEnvironment()
	configureProcess(command)
	stdout := newBoundedBuffer(spec.MaxOutputBytes)
	stderr := newBoundedBuffer(spec.MaxOutputBytes)
	command.Stdout = stdout
	command.Stderr = stderr

	if err := command.Start(); err != nil {
		sink.Error(spec.ID, "command-start", "command could not be started")
		return
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	var waitErr error
	select {
	case waitErr = <-done:
	case <-commandContext.Done():
		terminateProcess(command)
		<-done
		if ctx.Err() != nil {
			sink.Error(spec.ID, "deadline", "global collection deadline reached")
		} else {
			sink.Error(spec.ID, "timeout", "command timed out")
		}
	}
	if waitErr != nil && commandContext.Err() == nil {
		sink.Error(spec.ID, "command-exit", "command exited unsuccessfully")
	}
	addCommandOutput(sink, spec, "stdout.txt", stdout, sanitizer)
	addCommandOutput(sink, spec, "stderr.txt", stderr, sanitizer)
}

func addCommandOutput(sink *Sink, spec manifest.Collector, name string, output *boundedBuffer, sanitizer Sanitizer) {
	if output.buffer.Len() == 0 {
		return
	}
	if !isText(output.buffer.Bytes()) {
		sink.Error(spec.ID, "binary-skipped", "binary or unsupported command output skipped")
		return
	}
	clean, counts, err := sanitizer.Sanitize(output.buffer.Bytes())
	if err != nil {
		sink.Error(spec.ID, "redaction", "command output sanitization failed")
		return
	}
	if err := sink.Add(report.Entry{
		Name: "collectors/" + spec.ID + "/" + name, Data: clean,
		Collector: spec.ID, Truncated: output.truncated, Redactions: counts,
	}); err != nil {
		sink.Error(spec.ID, "limit", "bounded collection sink rejected command output")
	}
}

func max(value, floor int64) int64 {
	if value < floor {
		return floor
	}
	return value
}
