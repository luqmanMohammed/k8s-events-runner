package queue

type JobQueue chan *Job

func (jq *JobQueue) AddJob(job *Job) {
	*jq <- job
}

func NewJobQueue(queueSize int) JobQueue {
	return make(JobQueue, queueSize)
}
