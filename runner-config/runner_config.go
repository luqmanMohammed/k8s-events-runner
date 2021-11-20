package runnerconfig

import v1 "k8s.io/api/core/v1"

//RunnerConfig contains a k8s v1 api Pod definition
type RunnerConfig v1.Pod

//EventMapRunnerAssociation contains which runner
//should be used on event with aditional info
type EventMapRunnerAssociation struct {
	Runner           string `yaml:"runner"`
	ConcurrencyLimit int    `yaml:"concurrencyLimit" default:"1"`
	RetryLimit       int    `yaml:"retries" default:"0"`
}

//EventMapEventTypeAssociation associates Event type with specific runner association
type EventMapEventTypeAssociation map[string]EventMapRunnerAssociation

//EventMapResourceAssociation associates Resource with a specific event association
type EventMapResourceAssociation map[string]EventMapEventTypeAssociation
