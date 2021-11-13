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
	k8sconfigmapcollector "github.com/luqmanMohammed/k8s-events-runner/runner-config/config-collector/k8s-configmap-collector"
	"github.com/luqmanMohammed/k8s-events-runner/utils"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

type Config struct {
	Port           int
	LogLevel       string
	LogFormat      string
	Host           string
	IsLocal        bool
	KubeConfigPath string

	Namespace              string
	RunnerConfigMapLabel   string
	EventMapConfigMapLabel string
}

var (
	defaults = map[string]interface{}{
		"port":                 8080,
		"host":                 "0.0.0.0",
		"logLevel":             "debug",
		"logFormat":            "text",
		"isLocal":              true,
		"kubeConfigPath":       "",
		"namespace":            "er",
		"runnerConfigMapLabel": "er=runner",
	}
)

var rootCmd = &cobra.Command{
	Use:   "k8s-events-runner",
	Short: "Listens on events/requests and creates pods",
	Long:  `An automation tool which runs kubernets pods with configured inputs on configured events/requests`,
	Run: func(cmd *cobra.Command, args []string) {
		var config Config
		viper.Unmarshal(&config)
		kubeclientset, err := utils.GetKubeClientSet(config.IsLocal, config.KubeConfigPath)
		if err != nil {
			panic(err)
		}
		k8scmc := k8sconfigmapcollector.New(kubeclientset, config.Namespace, config.RunnerConfigMapLabel, config.EventMapConfigMapLabel)
		k8scmc.Collect()
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	for k, v := range defaults {
		viper.SetDefault(k, v)
	}
	viper.SetEnvPrefix("ER")
	viper.AutomaticEnv()
	rootCmd.Flags().IntP("port", "p", defaults["port"].(int), "Port to listen on")
	rootCmd.Flags().String("host", viper.GetString("host"), "Host to listen on")
	rootCmd.Flags().StringP("logLevel", "l", viper.GetString("logLevel"), "Log level")
	rootCmd.Flags().StringP("logFormat", "f", viper.GetString("logFormat"), "Log format")
	rootCmd.Flags().StringP("runnerEventMapConfigLabel", "e", viper.GetString("runnerEventMapConfigLabel"), "Label of runner event map config")
	rootCmd.Flags().BoolP("isLocal", "i", viper.GetBool("isLocal"), "Is local")
	rootCmd.Flags().StringP("kubeConfigPath", "k", viper.GetString("kubeConfigPath"), "Path to kube config")
	viper.BindPFlags(rootCmd.Flags())
}
