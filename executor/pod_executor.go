package executor

import (
	configcollector "github.com/luqmanMohammed/k8s-events-runner/runner-config/config-collector"
	"k8s.io/client-go/kubernetes"
)

type Job struct {
	configcollector.RunnerEventAssociation
	RetryCount int
}

type Executor interface {
}

type PodExecutor struct {
	namespace        string
	erPodIndentifier string
	k8sClientSet     *kubernetes.Clientset
}
