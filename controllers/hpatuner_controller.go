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

// +kubebuilder:rbac:groups=webapp.streamotion.com.au,resources=hpatuners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.streamotion.com.au,resources=hpatuners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=,resources=horizontalpodautoscalers,verbs=get;list;watch

func (r *HpaTunerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	rlog := r.Log.WithValues("hpatuner", req.NamespacedName)

	//hpatuner is a never ending forloop to keep on monitoring the hpa and action on it (until its deleted)
	resRepeat := reconcile.Result{RequeueAfter: r.syncPeriod}
	// resStop will be returned in case if we found some problem that can't be fixed, and we want to stop repeating reconcile process
	resStop := reconcile.Result{}

	log.Printf("start: ----------------------------------------------------------------------------------------------------") // to have clear separation between previous and current reconcile run
	log.Printf("")
	log.Printf("****Reconcile request: %v\n", req)

	// your logic here
	var hpaTuner webappv1.HpaTuner
	if err := r.Get(ctx, req.NamespacedName, &hpaTuner); err != nil {
		rlog.Error(err, "unable to fetch HpaTuner")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return resStop, client.IgnoreNotFound(err)
	}
	log.Printf("##: fetched %v \n", req.NamespacedName)

	//hpaRef := hpaTuner.Spec.ScaleTargetRef

	//log.Printf("hparef: %v \n", hpaRef)

	//TODO: check validity of hpaTuner

	hpaNamespace := hpaTuner.Namespace
	hpaName := hpaTuner.Spec.ScaleTargetRef.Name
	hpaNamespacedName := types.NamespacedName{Namespace: hpaNamespace, Name: hpaName}

	hpa := &scaleV1.HorizontalPodAutoscaler{}
	if err := r.Get(ctx, hpaNamespacedName, hpa); err != nil {
		// Error reading the object, repeat later
		rlog.Error(err, "Error reading HPA: ", "hpa", hpaNamespacedName)
		return resRepeat, nil
	}

	log.Printf("*Read HPA: %v", hpaNamespacedName)

	// --------------- now lets do reconcile.....
	if err := r.ReconcileHPA(&hpaTuner, hpa); err != nil {
		rlog.Error(err, "Could Not ReConcile")
		r.eventRecorder.Event(&hpaTuner, v1.EventTypeWarning, "FailedProcessHpaTuner", err.Error())

		return resStop, nil
	}
	// -----------------------------------------------------------------------------------
	log.Printf("") // to have clear separation between previous and current reconcile run
	log.Printf("--end: ----------------------------------------------------------------------------------------------------")

	// resRepeat will be returned if we want to re-run reconcile process
	// NB: we can't return non-nil err, as the "reconcile" msg will be added to the rate-limited queue
	// so that it'll slow down if we have several problems in a row

	return resRepeat, nil
}

// HpaTunerReconciler reconciles a HpaTuner object
func (r *HpaTunerReconciler) ReconcileHPA(hpaTuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) (err error) {
	log.Printf("--trying to reconcile..hpa: %v", toString(hpa))
	//MUST READ: https://engineering.pivotal.io/post/gp4k-kubebuilder-lessons/

	//??scaleUp??
	scaleTarget := scaleTo(hpaTuner, hpa)
	if scaleTarget > *hpa.Spec.MinReplicas {
		log.Printf("*** I am going to lock the hpa min now... %v", scaleTarget)
		r.UpdateHpaMin(hpaTuner, hpa, scaleTarget)

		//r.eventRecorder.Event(hpaTuner, v1.EventTypeNormal, "SuccessfulLockMin", fmt.Sprintf("Locked Min to %v", scaleTarget))
	} else if isInScaledState(hpaTuner, hpa) {
		log.Printf("HPA IS IN SCALED MODE!!!")
		if isMinLocked(hpaTuner, hpa) {
			log.Printf("HPA Min is Locked!!!")
			if shouldUnlockMin(hpaTuner, hpa) {
				log.Printf("Need to UnlockMin")
				r.UpdateHpaMin(hpaTuner, hpa, hpaTuner.Spec.MinReplicas)
			} else {
				log.Printf("----hpa locked but scaledown condition not met")
			}
		}
	} else {
		log.Printf("Nothing to do...")
	}

	return nil
}

func (r *HpaTunerReconciler) UpdateHpaMin(hpaTuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler, scaleTarget int32) (err error) {
	r.Log.Info("UpdateHpaMin: ", "scaleTarget", scaleTarget)
	hpa.Spec.MinReplicas = &scaleTarget
	if err := r.Client.Update(context.TODO(), hpa); err != nil {
		r.Log.Error(err, "Failed to Update hpa Min", "scaleTarget", scaleTarget)
	}

	hpaTuner.Status.LastUpScaleTime = &metav1.Time{}
	hpaTuner.Status.LastUpScaleTime.Time = time.Now() //TODO: put in constructor

	r.Client.Update(context.TODO(), hpaTuner)

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
	return hpa.Status.DesiredReplicas
}

func shouldUnlockMin(tuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) bool {
	downscaleForbiddenWindow := time.Duration(tuner.Spec.DownscaleForbiddenWindowSeconds) * time.Second
	log.Printf("-----: allowed scaledown: %v", hpa.Status.LastScaleTime.Add(downscaleForbiddenWindow))

	if elapsedDownscaleForbiddenWindow(hpa, downscaleForbiddenWindow) {
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

func elapsedDownscaleForbiddenWindow(hpa *scaleV1.HorizontalPodAutoscaler, downscaleForbiddenWindow time.Duration) bool {
	return hpa.Status.LastScaleTime != nil && hpa.Status.LastScaleTime.Add(downscaleForbiddenWindow).Before(time.Now())
}

func isMinLocked(tuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) bool {
	return *hpa.Spec.MinReplicas > tuner.Spec.MinReplicas
}

func isInScaledState(hpaTuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) bool {
	return hpaTuner.Spec.MinReplicas < hpa.Status.DesiredReplicas
}

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
