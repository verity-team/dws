package common

import (
	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
)

// RegisterJobErrorListener makes sure the errors returned by the scheduled jobs
// of s are logged together with the name of the job that failed.
//
// gocron hands the error a job returned to the "on error" event listener of
// that job and discards it silently when no such listener is registered.
//
// Please note: gocron registers the listener with the jobs that are known to
// the scheduler at the time of the call i.e. this function must be called
// *after* all jobs have been scheduled.
func RegisterJobErrorListener(s *gocron.Scheduler, service string) {
	s.RegisterEventListeners(gocron.WhenJobReturnsError(func(jobName string, err error) {
		log.Errorf("%s: scheduled job '%s' failed, %v", service, jobName, err)
	}))
}
