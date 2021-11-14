package k8sconfigmapcollector

import (
	"context"
	"encoding/json"
	"fmt"

	runnerconfig "github.com/luqmanMohammed/k8s-events-runner/runner-config"
	configcollector "github.com/luqmanMohammed/k8s-events-runner/runner-config/config-collector"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sConfigMapCollector struct {
	k8sClientSet          *kubernetes.Clientset
	namespace             string
	runnerConfigLable     string
	eventMapConfigMapName string
	runnerConfigs         map[string]*runnerconfig.RunnerConfig
	eventMap              runnerconfig.EventMapResourceAssociation
}

func New(k8sClientSet *kubernetes.Clientset, namespace, runnerConfigLable, eventMapConfigMapName string) *K8sConfigMapCollector {
	return &K8sConfigMapCollector{
		k8sClientSet:          k8sClientSet,
		namespace:             namespace,
		runnerConfigLable:     runnerConfigLable,
		eventMapConfigMapName: eventMapConfigMapName,
		runnerConfigs:         make(map[string]*runnerconfig.RunnerConfig),
	}
}

func (cmc *K8sConfigMapCollector) collectRunnerConfigs() error {
	runnerCMlist, err := cmc.k8sClientSet.CoreV1().ConfigMaps(cmc.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: cmc.runnerConfigLable,
	})
	if err != nil {
		return err
	}
	for _, cm := range runnerCMlist.Items {
		for _, value := range cm.Data {
			var podTemplate v1.Pod
			if err = json.Unmarshal([]byte(value), &podTemplate); err != nil {
				continue
			}
			tmpRunnerConfig := runnerconfig.RunnerConfig(podTemplate)
			cmc.runnerConfigs[cm.Name] = &tmpRunnerConfig
		}

	}
	return nil
}

func (cmc *K8sConfigMapCollector) collectEventMap() error {
	eventMapCM, err := cmc.k8sClientSet.CoreV1().ConfigMaps(cmc.namespace).Get(context.TODO(), cmc.eventMapConfigMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for _, value := range eventMapCM.Data {
		var eventMapConfig runnerconfig.EventMapResourceAssociation
		if err = yaml.Unmarshal([]byte(value), &eventMapConfig); err != nil {
			continue
		}
		cmc.eventMap = eventMapConfig
		break
	}
	return nil
}

func (cmc *K8sConfigMapCollector) Collect() error {
	if err := cmc.collectRunnerConfigs(); err != nil {
		return err
	}
	if err := cmc.collectEventMap(); err != nil {
		return err
	}
	return nil
}

func (cmc K8sConfigMapCollector) GetRunnerEventAssocForResourceAndEvent(resource, event string) (*configcollector.RunnerEventAssociation, error) {
	if resourceAssoc, found := cmc.eventMap[resource]; found {
		if eventTypeAssoc, found := resourceAssoc[event]; found {
			if runnerConfig, found := cmc.runnerConfigs[eventTypeAssoc.Runner]; found {
				return &configcollector.RunnerEventAssociation{
					RunnerConfig:   runnerConfig,
					EventMapConfig: eventTypeAssoc,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("%s:%s Not Found", resource, event)
}
