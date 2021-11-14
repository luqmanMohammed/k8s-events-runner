package configcollector

import (
	"errors"

	runnerconfig "github.com/luqmanMohammed/k8s-events-runner/runner-config"
)

var (
	ErrRunnerNotFound = errors.New("RunnerConfig not found for requested resource and event")
)

//RunnerEventAssociation combines eventMap with actual runner definition
type RunnerEventAssociation struct {
	runnerconfig.EventMapRunnerAssociation
	RunnerConfig *runnerconfig.RunnerConfig
}

//ConfigCollector interface should be implemented by all config collectors
type ConfigCollector interface {
	Collect() error
	GetRunnerEventAssocForResourceAndEvent(resource, event string) (RunnerEventAssociation, error)
}
