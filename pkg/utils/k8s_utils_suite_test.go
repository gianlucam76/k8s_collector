package utils_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	scheme    *runtime.Scheme
	env       *envtest.Environment
	k8sClient client.Client
)

var (
	cacheSyncBackoff = wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   1.5,
		Steps:    8,
		Jitter:   0.4,
	}
)

func TestK8sUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "K8sUtils Suite")
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")

	var err error
	scheme, err = setupScheme()
	Expect(err).To(BeNil())

	env = &envtest.Environment{
		Scheme:                scheme,
		ErrorIfCRDPathMissing: true,
	}

	_, err = env.Start()
	if err != nil {
		Expect(err).To(BeNil())
	}

	k8sClient, err = client.New(env.Config, client.Options{Scheme: scheme})
	Expect(err).To(BeNil())

})

func setupScheme() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		return nil, err
	}

	return s, nil
}

func waitForObject(ctx context.Context, c client.Client, obj client.Object) {
	// Makes sure the cache is updated with the new object
	objCopy := obj.DeepCopyObject().(client.Object)
	key := client.ObjectKeyFromObject(obj)
	if err := wait.ExponentialBackoff(
		cacheSyncBackoff,
		func() (done bool, err error) {
			if err := c.Get(ctx, key, objCopy); err != nil {
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}); err != nil {
		Expect(err).To(BeNil())
	}
}
