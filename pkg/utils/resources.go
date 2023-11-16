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
	"os"
	"path"
	"path/filepath"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v2"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/client"

	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
)

func (a *Collector) dumpResources(ctx context.Context, resource *Resource, logger logr.Logger) error {
	logger = logger.WithValues("gvk", fmt.Sprintf("%s:%s:%s", resource.Group, resource.Version, resource.Kind))
	logger.Info("collecting resources")

	gvk := schema.GroupVersionKind{
		Group:   resource.Group,
		Version: resource.Version,
		Kind:    resource.Kind,
	}

	dc := discovery.NewDiscoveryClientForConfigOrDie(a.restConfig)
	groupResources, err := restmapper.GetAPIGroupResources(dc)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	d := dynamic.NewForConfigOrDie(a.restConfig)

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		if apimeta.IsNoMatchError(err) {
			return nil
		}
		return err
	}

	resourceId := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: mapping.Resource.Resource,
	}

	options := metav1.ListOptions{}

	if len(resource.LabelFilters) > 0 {
		labelFilter := ""
		for i := range resource.LabelFilters {
			if labelFilter != "" {
				labelFilter += ","
			}
			f := resource.LabelFilters[i]
			if f.Operation == libsveltosv1alpha1.OperationEqual {
				labelFilter += fmt.Sprintf("%s=%s", f.Key, f.Value)
			} else {
				labelFilter += fmt.Sprintf("%s!=%s", f.Key, f.Value)
			}
		}

		options.LabelSelector = labelFilter
	}

	if resource.Namespace != "" {
		if options.FieldSelector != "" {
			options.FieldSelector += ","
		}
		options.FieldSelector += fmt.Sprintf("metadata.namespace=%s", resource.Namespace)
	}

	list, err := d.Resource(resourceId).List(ctx, options)
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("collected %d resources", len(list.Items)))
	for i := range list.Items {
		err = a.dumpObject(&list.Items[i], logger)
		if err != nil {
			return err
		}
	}

	return nil
}

// dumpObject is a helper function to generically dump resource definition
// given the resource reference and file path for dumping location.
func (a *Collector) dumpObject(resource client.Object, logger logr.Logger) error {
	// Do not store resource version
	resource.SetResourceVersion("")
	err := a.addTypeInformationToObject(resource)
	if err != nil {
		return err
	}

	logger = logger.WithValues("kind", resource.GetObjectKind().GroupVersionKind().Kind)
	logger = logger.WithValues("resource", fmt.Sprintf("%s %s",
		resource.GetNamespace(), resource.GetName()))

	if !resource.GetDeletionTimestamp().IsZero() {
		logger.Info("resource is marked for deletion. Do not collect it.")
	}

	resourceYAML, err := yaml.Marshal(resource)
	if err != nil {
		return err
	}

	metaObj, err := apimeta.Accessor(resource)
	if err != nil {
		return err
	}

	kind := resource.GetObjectKind().GroupVersionKind().Kind
	namespace := metaObj.GetNamespace()
	name := metaObj.GetName()

	resourceFilePath := path.Join(a.directory, "resources", namespace, kind, name+".yaml")
	err = os.MkdirAll(filepath.Dir(resourceFilePath), permission0755)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(resourceFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, permission0644)
	if err != nil {
		return err
	}
	defer f.Close()

	logger.Info(fmt.Sprintf("storing resource in %s", resourceFilePath))
	return os.WriteFile(f.Name(), resourceYAML, permission0600)
}

func (a *Collector) addTypeInformationToObject(obj client.Object) error {
	gvks, _, err := a.scheme.ObjectKinds(obj)
	if err != nil {
		return err
	}

	for _, gvk := range gvks {
		if gvk.Kind == "" {
			continue
		}
		if gvk.Version == "" || gvk.Version == runtime.APIVersionInternal {
			continue
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		break
	}

	return nil
}
