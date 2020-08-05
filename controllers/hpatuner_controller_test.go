package controllers

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	scaleV1 "k8s.io/api/autoscaling/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	webappv1 "hpa-tuner/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"log"
	// "hpa-tuner/controllers"
)

const timeout = time.Second * 60 * 5
const interval = time.Second * 2

var _ = Describe("HpatunerController Tests - Happy Paths", func() {
	logger := log.New(GinkgoWriter, "INFO: ", log.Lshortfile)
	ctx := context.Background()
	fetchedLoadGeneratorPod := &v12.Pod{}

	BeforeEach(func() {
		loadgenerator := &v12.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      "load-generator",
				Namespace: "phpload",
			},
		}

		k8sClient.Delete(ctx, loadgenerator)

		k8sClient.DeleteAllOf(ctx, &scaleV1.HorizontalPodAutoscaler{}, client.InNamespace("phpload"))
		k8sClient.DeleteAllOf(ctx, &webappv1.HpaTuner{}, client.InNamespace("phpload"))

	})

	AfterEach(func() {
		if fetchedLoadGeneratorPod.Generation != 0 {
			k8sClient.Delete(ctx, fetchedLoadGeneratorPod)
			k8sClient.DeleteAllOf(ctx, &scaleV1.HorizontalPodAutoscaler{}, client.InNamespace("phpload"))
			k8sClient.DeleteAllOf(ctx, &webappv1.HpaTuner{}, client.InNamespace("phpload"))
		}
	})

	Context("HpaTuner Controller Tests", func() {
		It("T1: Test HpaMin Is overridden by HpaTunerMin", func() {
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
			verifier := verifierCurry(hpaNamespacedName, timeout*10)

			verifier(fmt.Sprintf("verify hpa.min was upped to match that of hpatuner.min %v", _hpaTunerRef.Spec.MinReplicas),func(fetchedHpa *scaleV1.HorizontalPodAutoscaler) bool { //verify hpa.min was upped to match that of hpatuner.min
				log.Printf("testing ")
				return *fetchedHpa.Spec.MinReplicas == _hpaTunerRef.Spec.MinReplicas
			})

			//TODO: find better assert mechanism, the below will fail to which assert in those `&&` fails
			Eventually(func() bool { // event was published to hpatuner
				opts := v1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s,involvedObject.uid=%s", toCreateTuner.Name, toCreateTuner.Namespace, _hpaTunerRef.UID)}
				events, _ := clientSet.CoreV1().Events(toCreateTuner.Namespace).List(opts)
				log.Print(events.Items[0])
				return events != nil && events.Items != nil && len(events.Items) == 1 && events.Items[0].Reason == "SuccessfulUpscaleMin"
			}, timeout, interval).Should(BeTrue())

		})

		It("T3: Test Decision From Decision Service is Honored", func() {
			logger.Println("----------------start test-----------")

			fakeDecisionService.FakeDecision.MinReplicas = 13

			toCreateHpa := generateHpa()
			Expect(k8sClient.Create(ctx, &toCreateHpa)).Should(Succeed())
			toCreateTuner := generateHpaTuner()
			toCreateTuner.Spec.UseDecisionService = true

			Expect(k8sClient.Create(ctx, &toCreateTuner)).Should(Succeed())

			logger.Printf("hpaMin: %v , tunerMin: %v", *toCreateHpa.Spec.MinReplicas, toCreateTuner.Spec.MinReplicas)

			time.Sleep(time.Second * 5)

			//TODO: more asserts!
			//see: https://github.com/microsoft/k8s-cronjob-prescaler/blob/fc649b04493d2157a6ddc29a418a71eac8ec0c83/controllers/prescaledcronjob_controller_test.go#L187
			hpaNamespacedName := types.NamespacedName{Namespace: toCreateTuner.Namespace, Name: toCreateTuner.Spec.ScaleTargetRef.Name}

			verifier := verifierCurry(hpaNamespacedName, timeout*10)

			decision, _ := fakeDecisionService.scalingDecision("", 0, 0)
			verifier("verify hpa.min was upped to match that from decisionService 13", func(fetchedHpa *scaleV1.HorizontalPodAutoscaler) bool {
				return *fetchedHpa.Spec.MinReplicas == decision.MinReplicas
			})

			fakeDecisionService.FakeDecision.MinReplicas = 16
			verifier("verify hpa.min was changed again when the decision service gave different decision 16", func(fetchedHpa *scaleV1.HorizontalPodAutoscaler) bool { //
				decision, _ := fakeDecisionService.scalingDecision("", 0, 0)
				return *fetchedHpa.Spec.MinReplicas == decision.MinReplicas
			})

			////TODO: how to change kind to make HPA change desired count faster?? below takes too long as it waits for k8s to scale down `desiredCount` after hpa min is changed
			fakeDecisionService.FakeDecision.MinReplicas = 7
			verifier("verify hpa.min was changed again when the decision service gave different decision 7", func(fetchedHpa *scaleV1.HorizontalPodAutoscaler) bool { //
				decision, _ := fakeDecisionService.scalingDecision("", 0, 0)
				hpaDownScaled := *fetchedHpa.Spec.MinReplicas == decision.MinReplicas
				return hpaDownScaled
			})

			//*This takes forever to pass (k8s takes long time to downscale hpa)
			//verifier(func(fetchedHpa *scaleV1.HorizontalPodAutoscaler) bool {
			//	log.Printf("--: %v %v %v", fetchedHpa.Status.DesiredReplicas, fetchedHpa.Status.CurrentReplicas, fakeDecisionService.scalingDecision().MinReplicas)
			//	return fetchedHpa.Status.CurrentReplicas == fakeDecisionService.scalingDecision().MinReplicas
			//})
		})

		It("WIP: Test HpaMin Is changed and locked with desired", func() {
			logger.Println("----------------start test-----------")

			toCreateHpa := generateHpa()
			toCreateHpa.Spec.MinReplicas = new(int32)
			*toCreateHpa.Spec.MinReplicas =  1
			toCreateHpa.Spec.MaxReplicas = 15

			Expect(k8sClient.Create(ctx, &toCreateHpa)).Should(Succeed())

			toCreateTuner := generateHpaTuner()
			toCreateTuner.Spec.UseDecisionService = false
			toCreateTuner.Spec.MinReplicas = 1
			Expect(k8sClient.Create(ctx, &toCreateTuner)).Should(Succeed())

			logger.Printf("hpaMin: %v , tunerMin: %v", toCreateHpa.Spec.MinReplicas, toCreateTuner.Spec.MinReplicas)

			time.Sleep(time.Second * 5)

			loadGeneratorPod := generateLoadPod()
			Expect(k8sClient.Create(ctx, &loadGeneratorPod)).Should(Succeed()) //this starts the load

			Eventually(func() bool{
				podName := types.NamespacedName{Name: loadGeneratorPod.Name, Namespace: loadGeneratorPod.Namespace}

				err := k8sClient.Get(ctx, podName, fetchedLoadGeneratorPod)
				Expect(err).Should(BeNil())

				return fetchedLoadGeneratorPod.Status.ContainerStatuses != nil &&  fetchedLoadGeneratorPod.Status.ContainerStatuses[0].Ready == true
			}, timeout, interval).Should(BeTrue())

			hpaVerifier := verifierCurry(types.NamespacedName{Namespace: toCreateHpa.Namespace, Name:toCreateHpa.Name})

			hpaVerifier(fmt.Sprintf("ensure min replica goes up to %v", toCreateHpa.Spec.MaxReplicas), func(autoscaler *scaleV1.HorizontalPodAutoscaler) bool { //ensure hpa goes all the way up
				return *autoscaler.Spec.MinReplicas == toCreateHpa.Spec.MaxReplicas
			})

			err := k8sClient.Delete(ctx, fetchedLoadGeneratorPod)
			Expect(err).Should(BeNil())


			hpaVerifier(fmt.Sprintf("ensure min replica comes down to %v", toCreateTuner.Spec.MinReplicas ),func(autoscaler *scaleV1.HorizontalPodAutoscaler) bool { //it should come down when no load eventually
				return *autoscaler.Spec.MinReplicas == toCreateTuner.Spec.MinReplicas
			})

		})

	})
})

/**
Curry Function, a bit of functional voodo but nicely hides the details of hpa fetch and reduce duplication, so necessary evil
*/
func verifierCurry(name types.NamespacedName, optTimeout ...time.Duration) func(testname string, condition func(fetchedHpa *scaleV1.HorizontalPodAutoscaler) bool) {
	eventuallyTimeOut := timeout

	if len(optTimeout) > 0 {
		eventuallyTimeOut = optTimeout[0]
	}

	return func(testname string, condition func(autoscaler *scaleV1.HorizontalPodAutoscaler) bool) {
		Eventually(func() bool {
			ctx := context.Background()
			fetchedHpa := scaleV1.HorizontalPodAutoscaler{}
			err := k8sClient.Get(ctx, name, &fetchedHpa)
			Expect(err).Should(BeNil())

			log.Printf("--[%v]-- hpa for assertion:  currentMin:%v/currentDesired:%v/currentReplica:%v",testname, fetchedHpa.Status.CurrentReplicas, fetchedHpa.Status.DesiredReplicas, fetchedHpa.Status.CurrentReplicas)

			return condition(&fetchedHpa)
		}, eventuallyTimeOut, interval).Should(BeTrue())
	}
}

func generateLoadPod() v12.Pod {
	containers  := [1]v12.Container{}

	containers[0] = v12.Container{
		Name:                     "load-generator",
		Image:                    "busybox:1.32.0",
		Command:                  []string {"/bin/sh"},
		Args:                     []string{"-c","while true; do wget -q -O-  http://php-apache; done"},
	}

	var thePod = v12.Pod{
		TypeMeta: v1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "load-generator",
			Namespace: "phpload",
		},
		Spec: v12.PodSpec{
			Containers: []v12.Container{
				{
					Name:                     "load-generator",
					Image:                    "busybox",
					Command:                  []string {"/bin/sh"},
					Args:                     []string{"-c","while true; do wget -q -O-  http://php-apache; done"},
				},
			},
		},
	}
	return thePod
}


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
		DownscaleForbiddenWindowSeconds:             30,
		UpscaleForbiddenWindowAfterDownScaleSeconds: 600,
		ScaleUpLimitFactor:                          2,
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
