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
	libsveltosv1alpha1 "github.com/projectsveltos/libsveltos/api/v1alpha1"
)

// Resource indicates the type of resources to collect.
type Resource struct {
	// Namespace of the resource deployed in the Cluster.
	// Empty for resources scoped at cluster level.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Group of the resource deployed in the Cluster.
	Group string `json:"group"`

	// Version of the resource deployed in the Cluster.
	Version string `json:"version"`

	// Kind of the resource deployed in the Cluster.
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`

	// LabelFilters allows to filter resources based on current labels.
	LabelFilters []libsveltosv1alpha1.LabelFilter `json:"labelFilters,omitempty"`
}

// LogFilter allows to select which logs to collect
type Log struct {
	// Namespace of the pods deployed in the Cluster.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// LabelFilters allows to filter pods based on current labels.
	LabelFilters []libsveltosv1alpha1.LabelFilter `json:"labelFilters,omitempty"`

	// A relative time in seconds before the current time from which to collect logs.
	// If this value precedes the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned. Only one of sinceSeconds or sinceTime may be specified.
	// +optional
	SinceSeconds *int64 `json:"sinceSeconds,omitempty"`
}

// Configuration defines the instruction for collector
type Configuration struct {
	// Resources indicates what resorces to collect
	// +optional
	Resources []Resource `json:"resources,omitempty"`

	// Logs indicates what pods' log to collect
	// +optional
	Logs []Log `json:"logs,omitempty"`
}
