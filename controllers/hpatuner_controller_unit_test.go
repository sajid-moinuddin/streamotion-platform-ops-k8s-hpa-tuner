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

func TestReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	webappv1.AddToScheme(scheme)
	v1.AddToScheme(scheme)
	scheme.AddKnownTypes(schema.GroupVersion{})

	hpa := generateHpaForNames("test-svc", "test-ns")
	hpaTuner := generateHpaTunerForNames("test-svc", "test-ns")

	client := fake.NewFakeClientWithScheme(scheme, &hpa, &hpaTuner)
	testDecisionService = FakeScalingDecisionService{
		FakeDecision: &ScalingDecision{MinReplicas: 1},
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
	reconciler.Get(context.TODO(), types.NamespacedName{Name: "test-svc", Namespace: "test-ns"}, currentHpa)

	if *currentHpa.Spec.MinReplicas != 1 {
		t.Errorf("Expected 1 Min replica but got %v", *hpa.Spec.MinReplicas)
	}

	// set the scaling min to 5
	testDecisionService.FakeDecision.MinReplicas = 5

	request := reconcile.Request{types.NamespacedName{Namespace: "test-ns", Name: "test-svc"}}
	result, err := reconciler.Reconcile(request)

	if err != nil {
		t.Error(err)
	}
	println("result: " + result.RequeueAfter.String())

	reconciler.Get(context.TODO(), types.NamespacedName{Name: "test-svc", Namespace: "test-ns"}, currentHpa)
	if *currentHpa.Spec.MinReplicas != 5 {
		t.Errorf("Expected 5 Min replica but got %v", *hpa.Spec.MinReplicas)
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

func generateHpaTunerForNames(name string, namespace string) webappv1.HpaTuner {
	spec := webappv1.HpaTunerSpec{
		DownscaleForbiddenWindowSeconds:             30,
		UpscaleForbiddenWindowAfterDownScaleSeconds: 600,
		ScaleUpLimitFactor:                          2,
		ScaleTargetRef: webappv1.CrossVersionObjectReference{
			Kind: "HorizontalPodAutoscaler",
			Name: name,
		},
		MinReplicas:        5,
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
			LastUpScaleTime:   &metav1.Time{Time: time.Now().Add(time.Duration(-100000))},
			LastDownScaleTime: &metav1.Time{Time: time.Now().Add(time.Duration(-100000))},
		},
	}

	return tuner
}
