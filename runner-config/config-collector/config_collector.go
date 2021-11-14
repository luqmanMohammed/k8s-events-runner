package configcollector

import (
	runnerconfig "github.com/luqmanMohammed/k8s-events-runner/runner-config"
)

type RunnerEventAssociation struct {
	RunnerConfig   *runnerconfig.RunnerConfig
	EventMapConfig runnerconfig.EventMapRunnerAssociation
}

type ConfigCollector interface {
	Collect() error
	GetRunnerEventAssocForResourceAndEvent(resource, event string) (RunnerEventAssociation, error)
}
