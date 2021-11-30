package queue

import "github.com/luqmanMohammed/k8s-events-runner/config"

type Job struct {
	config.RunnerConfig
	EventType string
	Resource  string
}
