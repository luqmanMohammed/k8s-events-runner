package k8sconfigmapcollector

import (
	"context"
	"encoding/json"

	config "github.com/luqmanMohammed/k8s-events-runner/config"
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
	runnerTemplates       map[string]*config.RunnerTemplate
	eventMap              config.EventMap
}

//New instanciates a K8sConfigMapCollector object
func New(k8sClientSet *kubernetes.Clientset, namespace, runnerConfigLable, eventMapConfigMapName string) *K8sConfigMapCollector {
	return &K8sConfigMapCollector{
		k8sClientSet:          k8sClientSet,
		namespace:             namespace,
		runnerConfigLable:     runnerConfigLable,
		eventMapConfigMapName: eventMapConfigMapName,
		runnerTemplates:       make(map[string]*config.RunnerTemplate),
	}
}

//collectRunnerConfigs collects runner templates from config maps in the defined
//namespace which have the defined label.
//ConfigMap name is used as a key to store the runner template
//TODO: Add support for hot loading configs
func (cmc *K8sConfigMapCollector) collectRunnerTemplates(ctx context.Context) error {
	runnerCMlist, err := cmc.k8sClientSet.CoreV1().ConfigMaps(cmc.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: cmc.runnerConfigLable,
	})
	if err != nil {
		klog.Errorf("Error when collecting runner templates %v", err)
		return err
	}
	for _, cm := range runnerCMlist.Items {
		for key, value := range cm.Data {
			var podTemplate v1.Pod
			if err = json.Unmarshal([]byte(value), &podTemplate); err != nil {
				klog.V(1).ErrorS(err, "Failed to collect runner template from %s:%s. Continuing", cm.Name, key)
				continue
			}
			tmpRunnerTemplate := config.RunnerTemplate(v1.PodTemplateSpec{
				ObjectMeta: podTemplate.ObjectMeta,
				Spec:       podTemplate.Spec,
			})
			cmc.runnerTemplates[cm.Name] = &tmpRunnerTemplate
		}
		klog.V(2).Infof("Collected templates from ConfigMap: %s", cm.Name)
	}
	klog.V(1).Info("Succesffully collected Runner Templates from all ConfigMaps")
	return nil
}

//collectEventMap collects eventMap config data from a specific confimap selected
//by using provided configmap name and namespace
//TODO: Add support for loading eventMap from multi configmaps
//TODO: Add support for hot-loading configs
//TODO: Add config validation
func (cmc *K8sConfigMapCollector) collectEventMap(ctx context.Context) error {
	eventMapCM, err := cmc.k8sClientSet.CoreV1().ConfigMaps(cmc.namespace).Get(ctx, cmc.eventMapConfigMapName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error when collecting eventMap Config %v", err)
		return err
	}
	for _, value := range eventMapCM.Data {
		var eventMapConfig config.EventMap
		if err = yaml.Unmarshal([]byte(value), &eventMapConfig); err != nil {
			klog.Errorf("Unable to collect eventMap. Invalid Config: %v", err)
			return err
		}
		cmc.eventMap = eventMapConfig
		break
	}
	klog.V(1).Infof("Succesffully collected EventMap from ConfigMap: %s", eventMapCM.Name)
	return nil
}

//Collect wraps above collector methods to collect both runner and eventMap configs
func (cmc *K8sConfigMapCollector) Collect() error {
	if err := cmc.collectRunnerTemplates(context.Background()); err != nil {
		return err
	}
	if err := cmc.collectEventMap(context.Background()); err != nil {
		return err
	}
	return nil
}

//GetRunnerConfigForResourceAndEvent is a getter which retrieves a runner configuration provided the reosurce and event
//TODO: Add support for event specific small overides
func (cmc K8sConfigMapCollector) GetRunnerConfigForResourceAndEvent(resource, event string) (config.RunnerConfig, error) {
	if runnerSelec, ok := cmc.eventMap[resource][event]; ok {
		if runnerTemplate, ok := cmc.runnerTemplates[runnerSelec.Runner]; ok {
			return config.RunnerConfig{
				RunnerSelector: runnerSelec,
				RunnerTemplate: runnerTemplate,
			}, nil
		}
	}
	return config.RunnerConfig{}, config.ErrRunnerConfigNotFound
}
