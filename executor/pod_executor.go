package executor

import "k8s.io/client-go/kubernetes"

type Executor interface {
}

type PodExecutor struct {
	namespace        string
	erPodIndentifier string
	k8sClientSet     *kubernetes.Clientset
}
