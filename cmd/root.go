/*
Copyright © 2021 Luqman Mohammed m.luqman077@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"flag"
	"time"

	"github.com/luqmanMohammed/k8s-events-runner/api"
	"github.com/luqmanMohammed/k8s-events-runner/executor"
	"github.com/luqmanMohammed/k8s-events-runner/queue"
	k8sconfigmapcollector "github.com/luqmanMohammed/k8s-events-runner/runner-config/config-collector/k8s-configmap-collector"
	"github.com/luqmanMohammed/k8s-events-runner/utils"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/spf13/viper"
)

//Config struct used to unmarshall all events-runner related configs
type Config struct {
	//General
	LogVerbosity string
	//ER server related configs
	Addr           string
	CACertPath     string
	ServerCertPath string
	ServerKeyPath  string
	//Kubernetes general configs
	IsLocal        bool
	KubeConfigPath string
	Namespace      string
	//Kubernetes configmap collector related configs
	RunnerConfigMapLabel  string
	EventMapConfigMapName string
	//Kubernetes event executor related configs
	ExecutorPodIdentifier string
	ConcurrencyTimeout    time.Duration
}

var (
	defaults = map[string]interface{}{
		"addr":                  ":8080",
		"logVerbosity":          "3",
		"isLocal":               true,
		"kubeConfigPath":        "",
		"namespace":             "er",
		"runnerConfigMapLabel":  "er=runner",
		"eventMapConfigMapName": "er-eventmap",
		"caCertPath":            "./test_pki/ca/ca.crt",
		"serverCertPath":        "./test_pki/server/server.crt",
		"serverKeyPath":         "./test_pki/server/server.key",
		"ExecutorPodIdentifier": "er",
		"ConcurrencyTimeout":    time.Minute * 5,
	}
)

var rootCmd = &cobra.Command{
	Use:   "k8s-events-runner",
	Short: "Listens on events/requests and creates pods",
	Long:  `An automation tool which runs kubernets pods with configured inputs for configured events/requests`,
	Run: func(cmd *cobra.Command, args []string) {
		var config Config
		viper.Unmarshal(&config)
		klog.InitFlags(nil)
		defer klog.Flush()
		flag.Set("v", config.LogVerbosity)
		klog.Info("Starting Events Runner")
		klog.V(1).Info("Initializing Kube Connection")
		kubeclientset, err := utils.GetKubeClientSet(config.IsLocal, config.KubeConfigPath)
		if err != nil {
			klog.Fatalf("Error Initializing Kube Connection: %v", err)
		}
		k8scmc := k8sconfigmapcollector.New(kubeclientset, config.Namespace, config.RunnerConfigMapLabel, config.EventMapConfigMapName)
		if err = k8scmc.Collect(); err != nil {
			klog.Fatalf("Error collecting configmaps: %v", err)
		}
		klog.V(1).Info("Starting Events Runner Server")
		jq := queue.NewJobQueue(50)
		exec := executor.New(kubeclientset, config.Namespace, config.ExecutorPodIdentifier, jq)

		go func() {
			exec.StartExecutors(context.Background())
		}()

		erServer := api.New(config.Addr, &jq, k8scmc)
		if err = erServer.ListenMTLS(config.CACertPath, config.ServerKeyPath, config.ServerCertPath); err != nil {
			klog.Fatalf("Error starting server: %v", err)
		}
	},
}

//Execute triggers the root cmd
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	for k, v := range defaults {
		viper.SetDefault(k, v)
	}
	viper.SetEnvPrefix("ER")
	viper.AutomaticEnv()
	viper.BindPFlags(rootCmd.Flags())
}
