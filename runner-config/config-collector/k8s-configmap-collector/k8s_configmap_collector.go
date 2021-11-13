package k8sconfigmapcollector

import (
	"context"
	"encoding/json"
	"fmt"

	runnerconfig "github.com/luqmanMohammed/k8s-events-runner/runner-config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sConfigMapCollector struct {
	k8sClientSet        *kubernetes.Clientset
	namespace           string
	runnerConfigLable   string
	eventMapConfigLable string
	runnerConfigs       map[string]runnerconfig.RunnerConfig
}

func New(k8sClientSet *kubernetes.Clientset, namespace, runnerConfigLable, eventMapConfigLable string) *K8sConfigMapCollector {
	return &K8sConfigMapCollector{
		k8sClientSet:        k8sClientSet,
		namespace:           namespace,
		runnerConfigLable:   runnerConfigLable,
		eventMapConfigLable: eventMapConfigLable,
		runnerConfigs:       make(map[string]runnerconfig.RunnerConfig),
	}
}

func (cmc *K8sConfigMapCollector) Collect() error {
	runnerCMlist, err := cmc.k8sClientSet.CoreV1().ConfigMaps(cmc.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: cmc.runnerConfigLable,
	})
	if err != nil {
		return err
	}
	for _, cm := range runnerCMlist.Items {
		for key, value := range cm.Data {
			var podTemplate v1.Pod
			if err = json.Unmarshal([]byte(value), &podTemplate); err != nil {
				continue
			}
			cmc.runnerConfigs[key] = runnerconfig.RunnerConfig(podTemplate)
		}

	}
	fmt.Println(cmc.runnerConfigs)
	return nil
}
