package utils

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

//getKubeAPIConfig gets config based in-cluster or out-cluster
func getKubeAPIConfig(isLocal bool, kubeConfigPath string) (*rest.Config, error) {
	if isLocal {
		klog.V(3).Info("Client detected to be running in local")
		if kubeConfigPath == "" {
			klog.V(3).Info("Provided KubeConfig path is empty. Getting config from home")
			if home := homedir.HomeDir(); home != "" {
				kubeConfigPath = home + "/.kube/config"
			}
		}
		return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	} else {
		klog.V(3).Info("Initilizing incluster config")
		return rest.InClusterConfig()
	}
}

//GetKubeClientSet creates and returns ClientSet getting config dynamically
func GetKubeClientSet(isLocal bool, kubeConfigPath string) (*kubernetes.Clientset, error) {
	config, err := getKubeAPIConfig(isLocal, kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
