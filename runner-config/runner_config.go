package runnerconfig

import v1 "k8s.io/api/core/v1"

type RunnerConfig v1.Pod

type EventMapRunnerAssociation struct {
	Runner      string `yaml:"runner"`
	Concurrency int    `yaml:"concurrency"`
}
type EventMapEventTypeAssociation map[string]EventMapRunnerAssociation
type EventMapResourceAssociation map[string]EventMapEventTypeAssociation
