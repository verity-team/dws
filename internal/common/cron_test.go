package common

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// syncBuffer collects log output written by the scheduler goroutines
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func failingJob() error {
	return errors.New("job went sideways")
}

// an error returned by a scheduled job must not be discarded silently
func TestRegisterJobErrorListener(t *testing.T) {
	out := &syncBuffer{}
	orig := log.StandardLogger().Out
	log.SetOutput(out)
	defer log.SetOutput(orig)

	s := gocron.NewScheduler(time.UTC)
	s.SingletonModeAll()
	_, err := s.Every("100ms").Do(failingJob)
	assert.Nil(t, err)

	// the listener has to be registered *after* the job was scheduled
	RegisterJobErrorListener(s, "test-service")

	s.StartAsync()
	defer s.Stop()

	assert.Eventually(t, func() bool {
		return strings.Contains(out.String(), "job went sideways")
	}, 5*time.Second, 10*time.Millisecond, "the job error was never logged")

	logged := out.String()
	assert.Contains(t, logged, "test-service")
	assert.Contains(t, logged, "failingJob")
}

// without a registered listener gocron drops the error on the floor; this is
// the behaviour the listener above compensates for
func TestJobErrorsAreDiscardedWithoutListener(t *testing.T) {
	out := &syncBuffer{}
	orig := log.StandardLogger().Out
	log.SetOutput(out)
	defer log.SetOutput(orig)

	s := gocron.NewScheduler(time.UTC)
	s.SingletonModeAll()
	_, err := s.Every("100ms").Do(failingJob)
	assert.Nil(t, err)

	s.StartAsync()
	// wait for the job to have run at least once
	assert.Eventually(t, func() bool {
		return s.Jobs()[0].RunCount() > 0
	}, 5*time.Second, 10*time.Millisecond)
	s.Stop()

	assert.NotContains(t, out.String(), "job went sideways")
}
