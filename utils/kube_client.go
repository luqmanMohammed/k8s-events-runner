package utils

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func getKubeAPIConfig(isLocal bool, kubeConfigPath string) (*rest.Config, error) {

	if isLocal {
		if kubeConfigPath == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeConfigPath = home + "/.kube/config"
			}
		}
		return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	} else {
		return rest.InClusterConfig()
	}
}

func GetKubeClientSet(isLocal bool, kubeConfigPath string) (*kubernetes.Clientset, error) {
	config, err := getKubeAPIConfig(isLocal, kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
