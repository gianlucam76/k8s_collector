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

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (a *Collector) CollectResouces(ctx context.Context, logger logr.Logger) error {
	config, err := a.loadConfiguration(ctx, logger)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get configuration: %v", err))
		return err
	}

	if config == nil {
		logger.Info("no configuration present")
		return nil
	}

	err = a.collectData(ctx, config, logger)

	return err
}

func (a *Collector) loadConfiguration(ctx context.Context, logger logr.Logger) (*Configuration, error) {
	namespace := os.Getenv("COLLECTOR_NAMESPACE")

	logger = logger.WithValues("configmap", fmt.Sprintf("%s/%s", namespace, a.configMapName))
	logger.Info("getting ConfigMap ")

	configMap := &corev1.ConfigMap{}
	err := a.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: a.configMapName}, configMap)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get configMap: %v", err))
		return nil, err
	}

	if configMap.Data == nil {
		logger.Info("configMap Data is nil")
		return nil, fmt.Errorf("nil ConfigMap.Data")
	}

	for k := range configMap.Data {
		var currentConfiguration Configuration

		err = yaml.Unmarshal([]byte(configMap.Data[k]), &currentConfiguration)
		if err == nil {
			return &currentConfiguration, nil
		}

		err = json.Unmarshal([]byte(configMap.Data[k]), &currentConfiguration)
		if err == nil {
			return &currentConfiguration, nil
		}

		logger.Info(fmt.Sprintf("content %q", configMap.Data[k]))
		logger.Info(fmt.Sprintf("configMap key: %q does not contain a valid configuration instance: %v", k, err))
	}

	return nil, nil
}

func (a *Collector) collectData(ctx context.Context, configuration *Configuration, logger logr.Logger) error {
	logger.Info("collecting logs")
	var err error
	for i := range configuration.Logs {
		tmpErr := a.collectLogs(ctx, &configuration.Logs[i], logger)
		if tmpErr != nil {
			logger.Info(fmt.Sprintf("failed to collect logs %v", err))
			if err == nil {
				err = tmpErr
			} else {
				err = errors.Wrap(err, tmpErr.Error())
			}
		}
	}

	for i := range configuration.Resources {
		tmpErr := a.dumpResources(ctx, &configuration.Resources[i], logger)
		if tmpErr != nil {
			logger.Info(fmt.Sprintf("failed to dump resources %v", err))
			if err == nil {
				err = tmpErr
			} else {
				err = errors.Wrap(err, tmpErr.Error())
			}
		}
	}

	return err
}
