package jobqueue

import (
	"github.com/luqmanMohammed/k8s-events-runner/executor"
)

type JobQueue chan executor.Job

func (jq *JobQueue) AddJob(job executor.Job) {
	*jq <- job
}

func New(queueSize int) JobQueue {
	return make(JobQueue, queueSize)
}
