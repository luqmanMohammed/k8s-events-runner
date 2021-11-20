package queue

type WatchQueue chan *Job

func (wq *WatchQueue) AddJob(job Job) {
	*wq <- &job
}

func NewWatchQueue(queueSize int) WatchQueue {
	return make(WatchQueue, queueSize)
}
