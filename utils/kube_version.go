package utils

import (
	"strconv"

	"k8s.io/client-go/kubernetes"
)

func GetKubeVersion(clientSet *kubernetes.Clientset) (int, int, error) {
	versionInfo, err := clientSet.Discovery().ServerVersion()
	if err != nil {
		return 0, 0, err
	}
	intMajor, err := strconv.Atoi(versionInfo.Major)
	intMinor, err := strconv.Atoi(versionInfo.Minor)
	if err != nil {
		return 0, 0, err
	}
	return intMajor, intMinor, nil
}
