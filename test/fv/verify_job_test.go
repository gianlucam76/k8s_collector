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

package fv_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Verify Job", Label("FV"), func() {

	It("Job collects all resources and logs", func() {
		By("Wait for job to complete")
		Eventually(func() bool {
			job := &batchv1.Job{}

			err := k8sClient.Get(context.TODO(),
				types.NamespacedName{Namespace: "default", Name: "k8s-collector"}, job)
			if err != nil {
				return false
			}

			for i := range job.Status.Conditions {
				condition := &job.Status.Conditions[i]
				if condition.Type == batchv1.JobComplete &&
					condition.Status == corev1.ConditionTrue {
					return true
				}
			}

			return false
		}, timeout, pollingInterval).Should(BeTrue())
	})
})
