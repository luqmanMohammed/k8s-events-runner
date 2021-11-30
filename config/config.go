package config

import (
	"errors"

	v1 "k8s.io/api/core/v1"
)

var (
	ErrRunnerConfigNotFound = errors.New("RunnerConfig not found for requested resource and event")
)

//ConfigCollector interface should be implemented by all config collectors
type ConfigCollector interface {
	Collect() error
	GetRunnerConfigForResourceAndEvent(resource, event string) (RunnerConfig, error)
}

//RunnerTemplate is a template for a pod runner configuration
type RunnerTemplate v1.PodTemplateSpec

//EventMap maps and resource:event to a runner
type EventMap map[string]map[string]RunnerSelector

//RunnerSelector contains the runner name and event specific information
//TODO: Add simple event specific overides for runner configuration
type RunnerSelector struct {
	Runner           string `yaml:"runner"`
	ConcurrencyLimit int    `yaml:"concurrencyLimit" default:"-1"`
	RetryLimit       int    `yaml:"retryLimit" default:"0"`
}

//RunnerConfig contains actual runner template and event specific information
type RunnerConfig struct {
	RunnerSelector
	RunnerTemplate *RunnerTemplate
}
