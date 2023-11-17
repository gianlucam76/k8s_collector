package utils_test

import (
	"context"
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/textlogger"

	"github.com/gianlucam76/k8s_collector/pkg/utils"
)

var _ = Describe("Collect", func() {
	It("loadConfiguration loads configuration from ConfigMap (YAML)", func() {
		sinceSecond := int64(600)
		collectorConfig := &utils.Configuration{
			Logs: []utils.Log{
				{Namespace: "kube-system", SinceSeconds: &sinceSecond},
			},
			Resources: []utils.Resource{
				{Group: "", Version: "v1", Kind: "Secret"},
				{Group: "apps", Version: "v1", Kind: "Deployment"},
			},
		}

		jsonBytes, err := yaml.Marshal(collectorConfig)
		Expect(err).To(BeNil())

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "foo",
			},
			Data: map[string]string{
				"config": string(jsonBytes),
			},
		}

		collector, err := utils.GetCollectorInstance(scheme, env.Config, "", configMap.Name)
		Expect(err).To(BeNil())
		Expect(k8sClient.Create(context.TODO(), configMap)).To(Succeed())

		waitForObject(context.TODO(), k8sClient, configMap)

		os.Setenv("COLLECTOR_NAMESPACE", configMap.Namespace)
		config := textlogger.NewConfig(textlogger.Verbosity(1))
		logger := textlogger.NewLogger(config)
		collectorConfig, err = utils.LoadConfiguration(collector, context.TODO(), logger)
		Expect(err).To(BeNil())
		Expect(collectorConfig).ToNot(BeNil())
	})

	It("loadConfiguration loads configuration from ConfigMap (JSON)", func() {
		sinceSecond := int64(600)
		collectorConfig := &utils.Configuration{
			Logs: []utils.Log{
				{Namespace: "kube-system", SinceSeconds: &sinceSecond},
			},
			Resources: []utils.Resource{
				{Group: "", Version: "v1", Kind: "Service"},
				{Group: "", Version: "v1", Kind: "Pod"},
				{Group: "apps", Version: "v1", Kind: "Deployment"},
			},
		}

		jsonBytes, err := json.Marshal(collectorConfig)
		Expect(err).To(BeNil())

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "bar",
			},
			Data: map[string]string{
				"config": string(jsonBytes),
			},
		}

		collector, err := utils.GetCollectorInstance(scheme, env.Config, "", configMap.Name)
		Expect(err).To(BeNil())
		Expect(k8sClient.Create(context.TODO(), configMap)).To(Succeed())

		waitForObject(context.TODO(), k8sClient, configMap)

		os.Setenv("COLLECTOR_NAMESPACE", configMap.Namespace)
		config := textlogger.NewConfig(textlogger.Verbosity(1))
		logger := textlogger.NewLogger(config)
		collectorConfig, err = utils.LoadConfiguration(collector, context.TODO(), logger)
		Expect(err).To(BeNil())
		Expect(collectorConfig).ToNot(BeNil())
	})
})
