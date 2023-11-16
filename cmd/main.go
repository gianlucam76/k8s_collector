/*
Copyright 2023. projectsveltos.io. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gianlucam76/k8s_collector/pkg/utils"
	"github.com/spf13/pflag"
)

var (
	configMapName string
	directory     string
)

func main() {
	klog.InitFlags(nil)

	initFlags(pflag.CommandLine)
	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	config := textlogger.NewConfig(textlogger.Verbosity(1))
	logger := textlogger.NewLogger(config)

	if directory == "" {
		logger.Info("directory where to store logs and resources is not defined")
		panic(1)
	}

	ctx := context.Background()
	scheme, restConfig := initializeManagementClusterAccess()
	collector, err := utils.GetCollectorInstance(scheme, restConfig, directory, configMapName)
	if err != nil {
		logger.Info("failed to get collector instance: %v", err)
	}

	err = collector.CollectResouces(ctx, logger)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to collect data: %v", err))
		os.Exit(1)
	}
}

func initializeManagementClusterAccess() (*runtime.Scheme, *rest.Config) {
	scheme, err := getScheme()
	if err != nil {
		werr := fmt.Errorf("failed to get scheme %w", err)
		log.Fatal(werr)
	}

	restConfig := ctrl.GetConfigOrDie()
	restConfig.QPS = 100
	restConfig.Burst = 100

	return scheme, restConfig
}

func getScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

func initFlags(fs *pflag.FlagSet) {
	fs.StringVar(&configMapName,
		"config-map", "",
		"Name of the ConfigMap containing the configuration")

	fs.StringVar(&directory,
		"dir", "",
		"Name of the directory where logs and resources will be stored")
}
