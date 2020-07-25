/*


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

package controllers

import (
	"context"
	"fmt"
	"log"
	"time"

	"k8s.io/client-go/kubernetes/scheme"

	"github.com/go-logr/logr"
	"github.com/golang/glog"
	webappv1 "hpa-tuner/api/v1"
	scaleV1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultSyncPeriod                            = time.Second * 15
	defaultTargetCPUUtilizationPercentage  int32 = 80
	defaultTolerance                             = 0.1
	defaultDownscaleForbiddenWindowSeconds       = 300
	defaultUpscaleForbiddenWindowSeconds         = 300
	defaultScaleUpLimitMinimum                   = 4.0
	defaultScaleUpLimitFactor                    = 2.0
)

// HpaTunerReconciler reconciles a HpaTuner object
type HpaTunerReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	eventRecorder record.EventRecorder
	clientSet     kubernetes.Interface
	syncPeriod    time.Duration
}

//// HpaTunerStatus defines the observed state of HpaTuner
//type HpaTunerStatus struct {
//	// Last time a scaleup event was observed
//	LastUpScaleTime *metav1.Time `json:"lastUpScaleTime,omitempty"`
//	// Last time a scale-down event was observed
//	LastDownScaleTime *metav1.Time `json:"lastDownScaleTime,omitempty"`
//}
type DecisionServiceDecision struct {
	Decision *Decision `json:"decision,omitempty"`
}

type Decision struct {
	Number int32 `json:"number"`
}

// +kubebuilder:rbac:groups=webapp.streamotion.com.au,resources=hpatuners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.streamotion.com.au,resources=hpatuners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=,resources=horizontalpodautoscalers,verbs=get;list;watch

func (r *HpaTunerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("hpatuner", req.NamespacedName)

	//hpatuner is a never ending forloop to keep on monitoring the hpa and action on it (until its deleted)
	resRepeat := reconcile.Result{RequeueAfter: r.syncPeriod}
	// resStop will be returned in case if we found some problem that can't be fixed, and we want to stop repeating reconcile process
	resStop := reconcile.Result{}

	log.Info("Reconcile: ----------------------------------------------------------------------------------------------------") // to have clear separation between previous and current reconcile run

	// your logic here
	var hpaTuner webappv1.HpaTuner
	if err := r.Get(ctx, req.NamespacedName, &hpaTuner); err != nil {
		log.Error(err, "unable to fetch HpaTuner")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return resStop, client.IgnoreNotFound(err)
	}
	log.Info(fmt.Sprintf("##: fetched %v \n", req.NamespacedName))

	//TODO: check validity of hpaTuner

	hpaNamespace := hpaTuner.Namespace
	hpaName := hpaTuner.Spec.ScaleTargetRef.Name
	hpaNamespacedName := types.NamespacedName{Namespace: hpaNamespace, Name: hpaName}

	hpa := &scaleV1.HorizontalPodAutoscaler{}
	if err := r.Get(ctx, hpaNamespacedName, hpa); err != nil {
		// Error reading the object, repeat later
		log.Error(err, "Error reading HPA: ", "hpa", hpaNamespacedName)
		return resRepeat, nil
	}

	// --------------- ok so we got the hpa object & hpa-tuner object at hand, now lets do reconcile.....
	if err := r.ReconcileHPA(&hpaTuner, hpa); err != nil {
		log.Error(err, "Could Not ReConcile")
		r.eventRecorder.Event(&hpaTuner, v1.EventTypeWarning, "FailedProcessHpaTuner", err.Error())

		return resStop, nil
	}
	// -----------------------------------------------------------------------------------
	log.Info("") // to have clear separation between previous and current reconcile run
	log.Info("--end: ----------------------------------------------------------------------------------------------------")

	// resRepeat will be returned if we want to re-run reconcile process
	// NB: we can't return non-nil err, as the "reconcile" msg will be added to the rate-limited queue
	// so that it'll slow down if we have several problems in a row

	return resRepeat, nil
}

// HpaTunerReconciler reconciles a HpaTuner object
func (r *HpaTunerReconciler) ReconcileHPA(hpaTuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) (err error) {
	log := r.Log

	log.Info(fmt.Sprintf("--trying to reconcile..hpa: %v", toString(hpa)))

	//??scaleUp??
	scaleTarget := scaleTo(hpaTuner, hpa)

	log.Info(fmt.Sprintf("** reconcile -> currentHpaMin: %v <===> scaleTarget: %v", *hpa.Spec.MinReplicas, scaleTarget))

	if r.needsScalingHpaMin(scaleTarget, *hpa.Spec.MinReplicas) {
		log.Info(fmt.Sprintf("*** I am going to lock the hpa min now... %v", scaleTarget)) //debug
		r.UpdateHpaMin(hpaTuner, hpa, scaleTarget)
		r.eventRecorder.Event(hpaTuner, v1.EventTypeNormal, "SuccessfulLockMin", fmt.Sprintf("Locked Min to %v", scaleTarget))
	} else if isHpaMinAlreadyInScaledState(hpaTuner, hpa) {
		if canCoolDownHpaMin(hpaTuner, hpa) {
			log.Info("Need to UnlockMin")
			r.UpdateHpaMin(hpaTuner, hpa, hpaTuner.Spec.MinReplicas)
		} else {
			log.Info("----hpa locked but scaledown condition not met")
		}
	} else {
		log.V(1).Info("Nothing to do...")
	}

	return nil
}

func (r *HpaTunerReconciler) needsScalingHpaMin(scaleTarget int32, currntHpaMin int32) bool {
	return scaleTarget > currntHpaMin
}

func (r *HpaTunerReconciler) UpdateHpaMin(hpaTuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler, scaleTarget int32) (err error) {
	r.Log.Info("UpdateHpaMin: ", "scaleTarget", scaleTarget)
	hpa.Spec.MinReplicas = &scaleTarget
	if err := r.Client.Update(context.TODO(), hpa); err != nil {
		r.Log.Error(err, "Failed to Update hpa Min", "scaleTarget", scaleTarget)
	}

	hpaTuner.Status.LastUpScaleTime = &metav1.Time{Time: time.Now()}
	//hpaTuner.Status.LastUpScaleTime.Time = time.Now() //TODO: put in constructor

	if err := r.Client.Update(context.TODO(), hpaTuner); err != nil {
		r.Log.Error(err, "Failed to Update hpaTuner LastUpScaleTime", "scaleTarget", scaleTarget)
	}

	return nil
}

func toString(hpa *scaleV1.HorizontalPodAutoscaler) string {
	var lastScaleTime string

	if hpa.Status.LastScaleTime != nil {
		lastScaleTime = string(time.Since(hpa.Status.LastScaleTime.Time))
	} else {
		lastScaleTime = "NA"
	}

	var currCPU int32

	if hpa.Status.CurrentCPUUtilizationPercentage != nil {
		currCPU = *hpa.Status.CurrentCPUUtilizationPercentage
	} else {
		currCPU = 0
	}

	return fmt.Sprintf("n: %v, pod: %v/%v, cpu: %v/%v last:%v",
		hpa.Name,
		*hpa.Spec.MinReplicas,
		hpa.Status.DesiredReplicas,
		currCPU,
		*hpa.Spec.TargetCPUUtilizationPercentage,
		lastScaleTime)
}

func scaleTo(tuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) int32 {
	//I got control of the scale decision now!
	desiredReplicaFromDecisionService := getDesiredReplicaFromDecisionService(tuner, hpa)

	return max(tuner.Spec.MinReplicas, hpa.Status.DesiredReplicas, desiredReplicaFromDecisionService)
}

func max(nums ...int32) int32 {
	max := nums[0]
	for i := 1; i < len(nums); i++ {
		if max < nums[i] {
			max = nums[i]
		}
	}

	return max
}

func getDesiredReplicaFromDecisionService(tuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) int32 {
	//curl -X GET "http://localhost:8080/api/HorizontalPodAutoscaler?name=hpa-martian-content-qa&current-min=10&current-instance-count=5" -H "accept: application/json"

	return -1
}

func canCoolDownHpaMin(tuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) bool {
	if elapsedDownscaleForbiddenWindow(hpa, tuner) {
		//now I can consider letting it cooldown if idle
		if isIdle(hpa) {
			return true
		}
	}

	return false
}

func isIdle(hpa *scaleV1.HorizontalPodAutoscaler) bool {
	if *hpa.Status.CurrentCPUUtilizationPercentage < 5 {
		return true
	}
	return false
}

func elapsedDownscaleForbiddenWindow(hpa *scaleV1.HorizontalPodAutoscaler, tuner *webappv1.HpaTuner) bool {
	downscaleForbiddenWindow := time.Duration(tuner.Spec.DownscaleForbiddenWindowSeconds) * time.Second
	return tuner.Status.LastUpScaleTime != nil && tuner.Status.LastUpScaleTime.Add(downscaleForbiddenWindow).Before(time.Now())
}

func isHpaMinAlreadyInScaledState(hpaTuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) bool {
	return hpaTuner.Spec.MinReplicas < *hpa.Spec.MinReplicas
}

//TODO: need to/can use informer ?? otherwise will make get/list options directly to api , can put pressure on api-server, probably should be fine (api-server creates backpressure too)
func (r *HpaTunerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	clientConfig := mgr.GetConfig()
	clientSet, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	evtNamespacer := clientSet.CoreV1()
	broadcaster := record.NewBroadcaster()
	broadcaster.StartLogging(glog.Infof)
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: evtNamespacer.Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "hpa-tuner"})

	r.syncPeriod = defaultSyncPeriod
	r.clientSet = clientSet
	r.eventRecorder = recorder

	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.HpaTuner{}).
		Complete(r)
}
