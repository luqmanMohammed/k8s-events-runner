package queue

import configcollector "github.com/luqmanMohammed/k8s-events-runner/runner-config/config-collector"

type Job struct {
	configcollector.RunnerEventAssociation
	EventType string
	Resource  string
}
