package queue

import configcollector "github.com/luqmanMohammed/k8s-events-runner/runner-config/config-collector"

type Job struct {
	configcollector.RunnerEventAssociation
	RetryCount int
	EventType  string
	Resource   string
	Status     string
	PodName    string
}
