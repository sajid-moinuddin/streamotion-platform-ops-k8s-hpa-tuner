package controllers

import (
	"context"
	webappv1 "hpa-tuner/api/v1"
	v1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"
)

var testDecisionService FakeScalingDecisionService

func TestReconcileWithDecisionService(t *testing.T) {
	tests := map[string]struct {
		currentCount      int32
		scalingCount      int32
		lastScaledSeconds int32
		expectedCount     int32
	}{
		"upScale":                            {currentCount: 1, scalingCount: 3, lastScaledSeconds: 3600, expectedCount: 3},
		"upScaleIfWithinForbiddenWindow":     {currentCount: 1, scalingCount: 3, lastScaledSeconds: 1, expectedCount: 3},
		"downScale":                          {currentCount: 3, scalingCount: 1, lastScaledSeconds: 3600, expectedCount: 1},
		"noDownScaleIfWithinForbiddenWindow": {currentCount: 3, scalingCount: 1, lastScaledSeconds: 1, expectedCount: 3},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			webappv1.AddToScheme(scheme)
			v1.AddToScheme(scheme)
			scheme.AddKnownTypes(schema.GroupVersion{})

			sname := "test-svc"
			namespace := "test-ns"

			hpa := generateHpaForNames(sname, namespace)
			*hpa.Spec.MinReplicas = tc.currentCount
			hpaTuner := generateHpaTunerForNames(sname, namespace, tc.lastScaledSeconds)

			client := fake.NewFakeClientWithScheme(scheme, &hpa, &hpaTuner)
			testDecisionService = FakeScalingDecisionService{
				FakeDecision: &ScalingDecision{MinReplicas: tc.scalingCount},
			}

			recorder := record.NewFakeRecorder(100)

			reconciler := HpaTunerReconciler{
				Client:                 client,
				Log:                    TestLogger{T: t, LogInfo: false},
				Scheme:                 scheme,
				eventRecorder:          recorder,
				clientSet:              fake2.NewSimpleClientset(),
				syncPeriod:             time.Duration(1),
				scalingDecisionService: testDecisionService,
				k8sHpaDownScaleTime:    time.Duration(1),
			}
			currentHpa := &v1.HorizontalPodAutoscaler{}
			reconciler.Get(context.TODO(), types.NamespacedName{Name: sname, Namespace: namespace}, currentHpa)

			if *currentHpa.Spec.MinReplicas != tc.currentCount {
				t.Errorf("Expected %v Min replica but got %v", tc.currentCount, *currentHpa.Spec.MinReplicas)
			}

			request := reconcile.Request{types.NamespacedName{Namespace: namespace, Name: sname}}
			_, err := reconciler.Reconcile(request)

			if err != nil {
				t.Error(err)
			}

			reconciler.Get(context.TODO(), types.NamespacedName{Name: sname, Namespace: namespace}, currentHpa)
			if *currentHpa.Spec.MinReplicas != tc.expectedCount {
				t.Errorf("Expected %v Min replica but got %v", tc.expectedCount, *currentHpa.Spec.MinReplicas)
			}
		})
	}

}

func generateHpaForNames(name string, namespace string) v1.HorizontalPodAutoscaler {
	min := new(int32)
	*min = 1
	cpu := new(int32)
	*cpu = 20

	spec := v1.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: v1.CrossVersionObjectReference{
			Kind:       "Deployment",
			Name:       name,
			APIVersion: "apps/v1",
		},
		MinReplicas:                    min,
		MaxReplicas:                    20,
		TargetCPUUtilizationPercentage: cpu,
	}

	hpa := v1.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
		Status: v1.HorizontalPodAutoscalerStatus{
			ObservedGeneration:              nil,
			LastScaleTime:                   nil,
			CurrentReplicas:                 1,
			DesiredReplicas:                 1,
			CurrentCPUUtilizationPercentage: nil,
		},
	}
	return hpa
}

func generateHpaTunerForNames(name string, namespace string, lastScaled int32) webappv1.HpaTuner {
	spec := webappv1.HpaTunerSpec{
		DownscaleForbiddenWindowSeconds:             30,
		UpscaleForbiddenWindowAfterDownScaleSeconds: 30,
		ScaleUpLimitFactor:                          2,
		ScaleTargetRef: webappv1.CrossVersionObjectReference{
			Kind: "HorizontalPodAutoscaler",
			Name: name,
		},
		MinReplicas:        1,
		MaxReplicas:        1000,
		UseDecisionService: true,
	}

	tuner := webappv1.HpaTuner{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HpaTuner",
			APIVersion: "webapp.streamotion.com.au/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: spec,
		Status: webappv1.HpaTunerStatus{
			LastUpScaleTime:   &metav1.Time{Time: time.Now().Add(time.Duration(-1*lastScaled) * time.Second)},
			LastDownScaleTime: &metav1.Time{Time: time.Now().Add(time.Duration(-1*lastScaled) * time.Second)},
		},
	}

	return tuner
}
