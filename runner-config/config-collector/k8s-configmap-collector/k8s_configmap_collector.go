package k8sconfigmapcollector

import (
	"context"
	"encoding/json"

	runnerconfig "github.com/luqmanMohammed/k8s-events-runner/runner-config"
	configcollector "github.com/luqmanMohammed/k8s-events-runner/runner-config/config-collector"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

//K8sConfigMapCollector implents ConfigCollector interface and adds functionality
//to get configs from kuberenets config maps
type K8sConfigMapCollector struct {
	k8sClientSet          *kubernetes.Clientset
	namespace             string
	runnerConfigLable     string
	eventMapConfigMapName string
	runnerConfigs         map[string]*runnerconfig.RunnerConfig
	eventMap              runnerconfig.EventMapResourceAssociation
}

//New instanciates a K8sConfigMapCollector instance
func New(k8sClientSet *kubernetes.Clientset, namespace, runnerConfigLable, eventMapConfigMapName string) *K8sConfigMapCollector {
	return &K8sConfigMapCollector{
		k8sClientSet:          k8sClientSet,
		namespace:             namespace,
		runnerConfigLable:     runnerConfigLable,
		eventMapConfigMapName: eventMapConfigMapName,
		runnerConfigs:         make(map[string]*runnerconfig.RunnerConfig),
	}
}

//collectRunnerConfigs collects runner configs from config maps in the defined
//namespace which have the defined label
func (cmc *K8sConfigMapCollector) collectRunnerConfigs() error {
	runnerCMlist, err := cmc.k8sClientSet.CoreV1().ConfigMaps(cmc.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: cmc.runnerConfigLable,
	})
	if err != nil {
		klog.Errorf("Error when collecting runner Config %v", err)
		return err
	}
	for _, cm := range runnerCMlist.Items {
		for key, value := range cm.Data {
			var podTemplate v1.Pod
			if err = json.Unmarshal([]byte(value), &podTemplate); err != nil {
				klog.V(1).ErrorS(err, "Failed to collect runner config from %s:%s. Continuing", cm.Name, key)
				continue
			}
			tmpRunnerConfig := runnerconfig.RunnerConfig(podTemplate)
			cmc.runnerConfigs[cm.Name] = &tmpRunnerConfig
		}
		klog.V(2).Infof("Collected configs from ConfigMap: %s", cm.Name)
	}
	klog.V(1).Info("Succesffully collected Runner Configs")
	return nil
}

//collectEventMap collect eventMap config data from a specific confimap selected
//by using provided configmap name and namespace
func (cmc *K8sConfigMapCollector) collectEventMap() error {
	eventMapCM, err := cmc.k8sClientSet.CoreV1().ConfigMaps(cmc.namespace).Get(context.TODO(), cmc.eventMapConfigMapName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error when collecting eventMap Config %v", err)
		return err
	}
	for _, value := range eventMapCM.Data {
		var eventMapConfig runnerconfig.EventMapResourceAssociation
		if err = yaml.Unmarshal([]byte(value), &eventMapConfig); err != nil {
			klog.Errorf("Unable to collect eventMapcConfig. Invalid Config:%v", err)
			return err
		}
		cmc.eventMap = eventMapConfig
		break
	}
	klog.V(1).Infof("Succesffully collected EventMap Configs from ConfigMap: %s", eventMapCM.Name)
	return nil
}

//Collect wraps above collector methods to collect both runner and eventMap configs
func (cmc *K8sConfigMapCollector) Collect() error {
	if err := cmc.collectRunnerConfigs(); err != nil {
		return err
	}
	if err := cmc.collectEventMap(); err != nil {
		return err
	}
	return nil
}

//GetRunnerEventAssocForResourceAndEvent is a getter which retrieves a runner configuration provided the reosurce and event
func (cmc K8sConfigMapCollector) GetRunnerEventAssocForResourceAndEvent(resource, event string) (*configcollector.RunnerEventAssociation, error) {
	if resourceAssoc, found := cmc.eventMap[resource]; found {
		if eventTypeAssoc, found := resourceAssoc[event]; found {
			if runnerConfig, found := cmc.runnerConfigs[eventTypeAssoc.Runner]; found {
				return &configcollector.RunnerEventAssociation{
					EventMapRunnerAssociation: eventTypeAssoc,
					RunnerConfig:              runnerConfig,
				}, nil
			}
		}
	}
	klog.V(1).ErrorS(configcollector.ErrRunnerNotFound, "NOT FOUND", resource, event)
	return nil, configcollector.ErrRunnerNotFound
}
