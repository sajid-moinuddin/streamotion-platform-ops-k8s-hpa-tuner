package controllers

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	scaleV1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	const interval = time.Second * 2

	BeforeEach(func() {
		k8sClient.DeleteAllOf(ctx, &scaleV1.HorizontalPodAutoscaler{}, client.InNamespace("phpload"))
		k8sClient.DeleteAllOf(ctx, &webappv1.HpaTuner{}, client.InNamespace("phpload"))
	})

	AfterEach(func() {
		//if len(_testGeneratedResource.Namespace) > 0 {
		//	k8sClient.Delete(ctx, _testGeneratedResource)
		//}
	})

	Context("HpaTuner Controller Tests", func() {
		It("Test HpaMin Is overridden by HpaTunerMin", func() {
			logger.Println("----------------start test-----------")

			toCreateHpa := generateHpa()
			Expect(k8sClient.Create(ctx, &toCreateHpa)).Should(Succeed())
			toCreateTuner := generateHpaTuner()
			toCreateTuner.Spec.UseDecisionService = false
			Expect(k8sClient.Create(ctx, &toCreateTuner)).Should(Succeed())

			logger.Printf("hpaMin: %v , tunerMin: %v", toCreateHpa.Spec.MinReplicas, toCreateTuner.Spec.MinReplicas)

			time.Sleep(time.Second * 5)

			_hpaTunerRef := &webappv1.HpaTuner{}
			Eventually(func() bool { //verify hpatuner.lastUpScaleTime was updated
				err := k8sClient.Get(ctx, types.NamespacedName{Name: toCreateTuner.Name, Namespace: toCreateTuner.Namespace}, _hpaTunerRef)
				log.Printf("lastUpscaleTime: %v", _hpaTunerRef.Status.LastUpScaleTime)
				return err == nil && _hpaTunerRef.Status.LastUpScaleTime != nil
			}, timeout, interval).Should(BeTrue())

			//TODO: more asserts!
			//see: https://github.com/microsoft/k8s-cronjob-prescaler/blob/fc649b04493d2157a6ddc29a418a71eac8ec0c83/controllers/prescaledcronjob_controller_test.go#L187
			hpaNamespacedName := types.NamespacedName{Namespace: toCreateTuner.Namespace, Name: toCreateTuner.Spec.ScaleTargetRef.Name}

			hpa := &scaleV1.HorizontalPodAutoscaler{}

			Eventually(func() bool { //verify hpa.min was upped to match that of hpatuner.min
				if err := k8sClient.Get(ctx, hpaNamespacedName, hpa); err != nil {
					return false
				}
				log.Printf("waiting for condition hpaMin:%v=tunerMin:%v", *hpa.Spec.MinReplicas, _hpaTunerRef.Spec.MinReplicas)
				return *hpa.Spec.MinReplicas == _hpaTunerRef.Spec.MinReplicas
			}, timeout, interval).Should(BeTrue())

			//TODO: find better assert mechanism, the below will fail to which assert in those `&&` fails
			Eventually(func() bool { //verify event was published to hpatuner
				opts := v1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.uid=%s", toCreateTuner.Name, toCreateTuner.Namespace, _hpaTunerRef.UID)}
				events, _ := clientSet.CoreV1().Events(toCreateTuner.Namespace).List(opts)
				log.Print(events.Items[0])
				return events != nil && events.Items != nil && len(events.Items) == 1 && events.Items[0].Reason == "SuccessfulUpscaleMin"
			}, timeout, interval).Should(BeTrue())

		})

		//It("Test HpaMin Is changed and locked with desired", func() {
		//	logger.Println("----------------start test-----------")
		//
		//	toCreateHpa := generateHpa()
		//
		//	Expect(k8sClient.Create(ctx, &toCreateHpa)).Should(Succeed())
		//	toCreateTuner := generateHpaTuner()
		//	toCreateTuner.Spec.UseDecisionService = false
		//	toCreateTuner.Spec.MinReplicas = 1
		//	Expect(k8sClient.Create(ctx, &toCreateTuner)).Should(Succeed())
		//
		//	logger.Printf("hpaMin: %v , tunerMin: %v", toCreateHpa.Spec.MinReplicas, toCreateTuner.Spec.MinReplicas)
		//
		//	time.Sleep(time.Second * 5)
		//
		//	_hpaTunerRef := &webappv1.HpaTuner{}
		//	Eventually(func() bool { //verify hpatuner.lastUpScaleTime was updated
		//		err := k8sClient.Get(ctx, types.NamespacedName{Name: toCreateTuner.Name, Namespace: toCreateTuner.Namespace}, _hpaTunerRef)
		//		log.Printf("lastUpscaleTime: %v", _hpaTunerRef.Status.LastUpScaleTime)
		//		return err == nil && _hpaTunerRef.Status.LastUpScaleTime != nil
		//	}, timeout, interval).Should(BeTrue())
		//
		//	//TODO: more asserts!
		//	//see: https://github.com/microsoft/k8s-cronjob-prescaler/blob/fc649b04493d2157a6ddc29a418a71eac8ec0c83/controllers/prescaledcronjob_controller_test.go#L187
		//	hpaNamespacedName := types.NamespacedName{Namespace: toCreateTuner.Namespace, Name: toCreateTuner.Spec.ScaleTargetRef.Name}
		//
		//	hpa := &scaleV1.HorizontalPodAutoscaler{}
		//
		//	Eventually(func() bool { //verify hpa.min was upped to match that of hpatuner.min
		//		if err := k8sClient.Get(ctx, hpaNamespacedName, hpa); err != nil {
		//			return false
		//		}
		//		log.Printf("waiting for condition hpaMin:%v=tunerMin:%v", *hpa.Spec.MinReplicas, _hpaTunerRef.Spec.MinReplicas)
		//		return *hpa.Spec.MinReplicas == _hpaTunerRef.Spec.MinReplicas
		//	}, timeout, interval).Should(BeTrue())
		//
		//	//TODO: find better assert mechanism, the below will fail to which assert in those `&&` fails
		//	Eventually(func() bool { //verify event was published to hpatuner
		//		opts := v1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.uid=%s", toCreateTuner.Name, toCreateTuner.Namespace, _hpaTunerRef.UID)}
		//		events, _ := clientSet.CoreV1().Events(toCreateTuner.Namespace).List(opts)
		//		log.Print(events.Items[0])
		//		return events != nil && events.Items != nil && len(events.Items) == 1 && events.Items[0].Reason == "SuccessfulUpscaleMin"
		//	}, timeout, interval).Should(BeTrue())
		//
		//})

		It("Test Decision From Decision Service is Honored", func() {
			logger.Println("----------------start test-----------")

			fakeDecisionService.FakeDecision.MinReplicas = 13

			toCreateHpa := generateHpa()
			Expect(k8sClient.Create(ctx, &toCreateHpa)).Should(Succeed())
			toCreateTuner := generateHpaTuner()
			toCreateTuner.Spec.UseDecisionService = true

			Expect(k8sClient.Create(ctx, &toCreateTuner)).Should(Succeed())

			logger.Printf("hpaMin: %v , tunerMin: %v", toCreateHpa.Spec.MinReplicas, toCreateTuner.Spec.MinReplicas)

			time.Sleep(time.Second * 5)

			//TODO: more asserts!
			//see: https://github.com/microsoft/k8s-cronjob-prescaler/blob/fc649b04493d2157a6ddc29a418a71eac8ec0c83/controllers/prescaledcronjob_controller_test.go#L187
			hpaNamespacedName := types.NamespacedName{Namespace: toCreateTuner.Namespace, Name: toCreateTuner.Spec.ScaleTargetRef.Name}

			hpa := &scaleV1.HorizontalPodAutoscaler{}

			Eventually(func() bool { //verify hpa.min was upped to match that from decisionService
				if err := k8sClient.Get(ctx, hpaNamespacedName, hpa); err != nil {
					return false
				}
				log.Printf("waiting for condition hpaMin:%v=decisionServiceMin:%v", *hpa.Spec.MinReplicas, fakeDecisionService.scalingDecision().MinReplicas)
				return *hpa.Spec.MinReplicas == fakeDecisionService.scalingDecision().MinReplicas
			}, timeout, interval).Should(BeTrue())

			fakeDecisionService.FakeDecision.MinReplicas = 16
			Eventually(func() bool { //verify hpa.min was changed again when the decision service gave different decision
				if err := k8sClient.Get(ctx, hpaNamespacedName, hpa); err != nil {
					return false
				}
				log.Printf("waiting for condition hpaMin:%v=decisionServiceMin:%v", *hpa.Spec.MinReplicas, fakeDecisionService.scalingDecision().MinReplicas)
				return *hpa.Spec.MinReplicas == fakeDecisionService.scalingDecision().MinReplicas
			}, timeout, interval).Should(BeTrue())

			////TODO: how to change kind to make HPA change desired count faster?? below takes too long as it waits for k8s to scale down `desiredCount` after hpa min is changed
			fakeDecisionService.FakeDecision.MinReplicas = 7
			Eventually(func() bool { //verify hpa.min was changed again when the decision service gave different decision
				if err := k8sClient.Get(ctx, hpaNamespacedName, hpa); err != nil {
					return false
				}
				log.Printf("waiting for condition hpaMin:%v=decisionServiceMin:%v=desired:%v", *hpa.Spec.MinReplicas, fakeDecisionService.scalingDecision().MinReplicas,  hpa.Status.DesiredReplicas)
				return *hpa.Spec.MinReplicas == fakeDecisionService.scalingDecision().MinReplicas && hpa.Status.DesiredReplicas == fakeDecisionService.scalingDecision().MinReplicas
			}, timeout*10, //need to wait for desired to come down
				interval).Should(BeTrue())

		})

	})
})

func generateHpa() scaleV1.HorizontalPodAutoscaler {
	//TODO: ?? nicer way to initiate those *int32 ??
	min := new(int32)
	*min = 1
	cpu := new(int32)
	*cpu = 20

	spec := scaleV1.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: scaleV1.CrossVersionObjectReference{
			Kind:       "Deployment",
			Name:       "php-apache",
			APIVersion: "apps/v1",
		},
		MinReplicas:                    min,
		MaxReplicas:                    20,
		TargetCPUUtilizationPercentage: cpu,
	}

	hpa := scaleV1.HorizontalPodAutoscaler{
		TypeMeta: v1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "php-apache",
			Namespace: "phpload",
		},
		Spec: spec,
	}

	return hpa
}

func generateHpaTuner() webappv1.HpaTuner {
	spec := webappv1.HpaTunerSpec{
		DownscaleForbiddenWindowSeconds: 30,
		UpscaleForbiddenWindowAfterDownScaleSeconds: 600,
		ScaleUpLimitFactor:              2,
		ScaleTargetRef: webappv1.CrossVersionObjectReference{
			Kind: "HorizontalPodAutoscaler",
			Name: "php-apache",
		},
		MinReplicas:        5,
		MaxReplicas:        1000,
		UseDecisionService: false,
	}

	tuner := webappv1.HpaTuner{
		TypeMeta: v1.TypeMeta{
			Kind:       "HpaTuner",
			APIVersion: "webapp.streamotion.com.au/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "php-apache-tuner",
			Namespace: "phpload",
		},
		Spec: spec,
	}

	return tuner
}
