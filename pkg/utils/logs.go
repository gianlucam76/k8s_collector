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
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
)

const (
	permission0600 = 0600
	permission0644 = 0644
	permission0755 = 0755
)

func (a *Collector) collectLogs(ctx context.Context, log *Log, logger logr.Logger) error {
	options := client.ListOptions{}

	if len(log.LabelFilters) > 0 {
		labelFilter := ""
		for i := range log.LabelFilters {
			if labelFilter != "" {
				labelFilter += ","
			}
			f := log.LabelFilters[i]
			if f.Operation == libsveltosv1alpha1.OperationEqual {
				labelFilter += fmt.Sprintf("%s=%s", f.Key, f.Value)
			} else {
				labelFilter += fmt.Sprintf("%s!=%s", f.Key, f.Value)
			}
		}

		parsedSelector, err := labels.Parse(labelFilter)
		if err != nil {
			return err
		}
		options.LabelSelector = parsedSelector
	}

	if log.Namespace != "" {
		options.Namespace = log.Namespace
	}

	pods := &corev1.PodList{}
	if err := a.client.List(ctx, pods, &options); err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("found %d pods", len(pods.Items)))
	for i := range pods.Items {
		if err := a.dumpPodLogs(ctx, log.SinceSeconds, &pods.Items[i]); err != nil {
			return err
		}
	}

	return nil
}

// dumpPodLogs collects logs for all containers in a pod and store them.
// If pod has restarted, it will try to collect log from previous run as well.
func (a *Collector) dumpPodLogs(ctx context.Context, since *int64, pod *corev1.Pod) error {
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		resourceFilePath := path.Join(a.directory, "logs", pod.Namespace, pod.Name+"-"+container.Name)
		err := os.MkdirAll(filepath.Dir(resourceFilePath), permission0755)
		if err != nil {
			return err
		}

		err = a.collectPodLogs(ctx, pod.Namespace, pod.Name, container.Name, resourceFilePath, since, false)
		if err != nil {
			return err
		}

		// If container restarted, collect previous logs as well
		for i := range pod.Status.ContainerStatuses {
			containerStatus := &pod.Status.ContainerStatuses[i]
			if containerStatus.Name == container.Name &&
				containerStatus.RestartCount > 0 {

				resourceFilePath := path.Join(a.directory, "logs", pod.Namespace,
					pod.Name+"-"+container.Name+".previous")

				err := os.MkdirAll(filepath.Dir(resourceFilePath), permission0755)
				if err != nil {
					return err
				}

				err = a.collectPodLogs(ctx, pod.Namespace, pod.Name, container.Name, resourceFilePath, since, true)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// collectPodLogs collect logs for a given namespace/pod container
func (a *Collector) collectPodLogs(ctx context.Context, namespace, podName, containerName, filename string,
	since *int64, previous bool) (err error) {

	// open output file
	var fo *os.File
	fo, err = os.Create(filename)
	if err != nil {
		return err
	}
	// close fo on exit and check for its returned error
	defer func() {
		if cerr := fo.Close(); cerr != nil {
			if err == nil {
				err = cerr
			}
		}
	}()

	podLogOpts := corev1.PodLogOptions{}
	if containerName != "" {
		podLogOpts.Container = containerName
	}

	if previous {
		podLogOpts.Previous = previous
	}

	if since != nil {
		podLogOpts.SinceSeconds = since
	}

	req := a.clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)
	var podLogs io.ReadCloser
	podLogs, err = req.Stream(ctx)
	if err != nil {
		return err
	}
	defer podLogs.Close()

	_, err = io.Copy(fo, podLogs)

	return err
}
