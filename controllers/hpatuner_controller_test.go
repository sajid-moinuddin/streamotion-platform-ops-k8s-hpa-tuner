package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	scaleV1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	webappv1 "hpa-tuner/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"log"

	// "hpa-tuner/controllers"
)

var _ = Describe("HpatunerController Tests - Happy Paths", func() {
	logger := log.New(GinkgoWriter, "INFO: ", log.Lshortfile)
	ctx := context.Background()
	const timeout = time.Second * 60
	const interval = time.Second * 1
	_testGeneratedResource := &webappv1.HpaTuner{}

	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
	})

	AfterEach(func() {
		if len(_testGeneratedResource.Namespace) > 0 {
			k8sClient.Delete(ctx, _testGeneratedResource)
		}
	})

	Context("HpaTuner Controller Tests", func() {
		It("Should Be Able to create HpaTuner CRD", func() {
			logger.Println("----------------start test-----------")
			toCreate := generateHpaTuner()
			Expect(k8sClient.Create(ctx, &toCreate)).Should(Succeed())

			time.Sleep(time.Second * 5)

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: toCreate.Name, Namespace: toCreate.Namespace}, _testGeneratedResource)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			//TODO: more asserts!
			//see: https://github.com/microsoft/k8s-cronjob-prescaler/blob/fc649b04493d2157a6ddc29a418a71eac8ec0c83/controllers/prescaledcronjob_controller_test.go#L187
			hpaNamespacedName := types.NamespacedName{Namespace: _testGeneratedResource.Namespace, Name: _testGeneratedResource.Spec.ScaleTargetRef.Name}

			hpa := &scaleV1.HorizontalPodAutoscaler{}

			Eventually(func() bool {
				if err := k8sClient.Get(ctx,hpaNamespacedName, hpa); err != nil {
					return  false
				}
				log.Printf("********************************************hpa....%v=%v", *hpa.Spec.MinReplicas, _testGeneratedResource.Spec.MinReplicas)
				return *hpa.Spec.MinReplicas == _testGeneratedResource.Spec.MinReplicas
			}, timeout, interval).Should(BeTrue())

		})
	})
})

func generateHpaTuner() webappv1.HpaTuner {
	spec := webappv1.HpaTunerSpec{
		DownscaleForbiddenWindowSeconds: 50,
		ScaleUpLimitFactor: 2,
		ScaleTargetRef: webappv1.CrossVersionObjectReference{
			Kind: "HorizontalPodAutoscaler",
			Name: "php-apache",
		},
		MinReplicas: 5,
		MaxReplicas: 1000,

	}

	tuner := webappv1.HpaTuner{
		TypeMeta:   v1.TypeMeta{
			Kind: "HpaTuner",
			APIVersion: "webapp.streamotion.com.au/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: "php-apache-tuner",
			Namespace: "phpload",
		},
		Spec:       spec,
	}

	return tuner
}