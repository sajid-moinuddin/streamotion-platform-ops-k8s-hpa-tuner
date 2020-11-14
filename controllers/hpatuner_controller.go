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
	"errors"
	"fmt"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"time"

	"k8s.io/client-go/kubernetes/scheme"

	"github.com/go-logr/logr"
	webappv1 "hpa-tuner/api/v1"
	scaleV1 "k8s.io/api/autoscaling/v1"
	"k8s.io/api/core/v1"
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
	Log                    logr.Logger
	Scheme                 *runtime.Scheme
	eventRecorder          record.EventRecorder
	clientSet              kubernetes.Interface
	syncPeriod             time.Duration
	scalingDecisionService ScalingDecisionService
	k8sHpaDownScaleTime    time.Duration //time takes for k8s to change desired count when cpu is idle

}

// +kubebuilder:rbac:groups=webapp.streamotion.com.au,resources=hpatuners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.streamotion.com.au,resources=hpatuners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=,resources=horizontalpodautoscalers,verbs=get;list;watch

func (r *HpaTunerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	/*template method to hide k8s controller details, main calculation is delegated after k8s objects are fetched*/

	ctx := context.Background()
	log := r.Log.WithValues("hpatuner", req.NamespacedName)

	//hpatuner is a never ending forloop to keep on monitoring the hpa and action on it (until its deleted)
	resRepeat := reconcile.Result{RequeueAfter: r.syncPeriod}
	// resStop will be returned in case if we found some problem that can't be fixed, and we want to stop repeating reconcile process
	resStop := reconcile.Result{}

	log.Info("********************* START RECONCILE **********************") // to have clear separation between previous and current reconcile run
	defer log.Info("********************* FINISHED RECONCILE **********************") // to have clear separation between previous and current reconcile run

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
	log := r.Log.WithValues("hpatuner", hpaTuner.Name)

	log.Info("**** Rconcile........", "hpa: " , toString(hpa), ", tuner: ", toStringTuner(*hpaTuner))

	decisionServiceDesired := r.getDesiredReplicaFromDecisionService(hpaTuner, hpa)
	needsScaling, scalingTarget := r.determineScalingNeeds(hpaTuner, hpa, decisionServiceDesired)

	log.Info("***Reconcile: ", "hpa", toString(hpa), "tuner: ", toStringTuner(*hpaTuner), "useDecision", hpaTuner.Spec.UseDecisionService, "decisionServiceDesired", decisionServiceDesired, "needsScaling: ", needsScaling, "scalingTarget", scalingTarget)

	if needsScaling {
		log.Info(fmt.Sprintf("*** I am going to lock the hpa min now... %v", scalingTarget)) //debug
		updated, _ := r.UpdateHpaMin(hpaTuner, hpa, scalingTarget)
		if updated {
			r.eventRecorder.Event(hpaTuner, v1.EventTypeNormal, "SuccessfulUpscaleMin", fmt.Sprintf("Locked Min to %v", scalingTarget))
		}
	} else if isHpaMinAlreadyInScaledState(hpaTuner, hpa) {
		if canCoolDownHpaMin(hpaTuner, hpa, decisionServiceDesired) {
			downscaleTarget := max(hpaTuner.Spec.MinReplicas, decisionServiceDesired)

			if downscaleTarget == *hpa.Spec.MinReplicas {
				log.V(2).Info("no action needed")
			} else {
				log.Info("Need to UnlockMin")

				updated, _ := r.UpdateHpaMin(hpaTuner, hpa, downscaleTarget) //decision service always wins
				if updated {
					r.eventRecorder.Event(hpaTuner, v1.EventTypeNormal, "SuccessfulDownscaleMin", fmt.Sprintf("Locked Min to %v", downscaleTarget))
				}
			}
		} else {
			log.Info("----hpa locked but scaledown condition not met", "elapsed: ", elapsedDownscaleForbiddenWindow(hpa, hpaTuner), "isIdle: ", isIdle(hpa))
		}
	} else {
		log.V(1).Info("Nothing to do...")
	}

	return nil
}

func (r *HpaTunerReconciler) getDesiredReplicaFromDecisionService(tuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) int32 {
	//curl -X GET "http://localhost:8080/api/HorizontalPodAutoscaler?name=hpa-martian-content-qa&current-min=10&current-instance-count=5" -H "accept: application/json"

	if tuner.Spec.UseDecisionService && r.scalingDecisionService == nil {
		r.Log.Error(errors.New("Null Decision Service!!!"), fmt.Sprintf("Wants to use decision service but decisionservice is nil! %v", tuner.Name))
		return -1
	}

	hpaName := types.NamespacedName{Name: hpa.Name, Namespace: hpa.Namespace}.String()

	if tuner.Spec.UseDecisionService {
		decision, err := r.scalingDecisionService.scalingDecision(hpaName, *hpa.Spec.MinReplicas, hpa.Status.CurrentReplicas)

		if err != nil {
			r.Log.Error(err, "failed to fetch result from decisionservice")
			return -1
		}

		r.Log.Info("Received From Decision Service: ", "minReplica: ", decision.MinReplicas)
		return decision.MinReplicas
	} else {
		r.Log.Info("Not using decision service") //todo: debug
	}

	return -1
}

func (r *HpaTunerReconciler) determineScalingNeeds(tuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler, decisionServiceDesired int32) (bool, int32) {
	currentDesired := hpa.Status.DesiredReplicas
	currentHpaMin := *hpa.Spec.MinReplicas
	actualMin := tuner.Spec.MinReplicas

	if r.recentlyDownScaled(tuner) { //if recently downscaled, ignore the desired counts

		if currentHpaMin < decisionServiceDesired {
			r.Log.V(1).Info("Not skipping upscale check...",
				"hpa", toString(hpa),
				"lastDownscaled", tuner.Status.LastDownScaleTime)
			return true, decisionServiceDesired
		} else {
			r.Log.V(1).Info("Skipping upscale check as it was recently downscaled..",
				"hpa", toString(hpa),
				"lastDownscaled", tuner.Status.LastDownScaleTime)
			return false, 0
		}
	} else {
		newMax := max(decisionServiceDesired, actualMin, currentDesired)

		if newMax > currentHpaMin {
			return true, newMax
		} else {
			return false, 0
		}
	}

}

func (r *HpaTunerReconciler) recentlyDownScaled(tuner *webappv1.HpaTuner) bool {
	upscaleForbiddenWindow := time.Duration(tuner.Spec.UpscaleForbiddenWindowAfterDownScaleSeconds) * time.Second
	r.Log.V(1).Info("Checking if recently downscaled: ", "upscaleForbiddenWindow", upscaleForbiddenWindow, "lastDownscaleTime", tuner.Status.LastDownScaleTime)

	if tuner.Status.LastDownScaleTime != nil && tuner.Status.LastDownScaleTime.Add(upscaleForbiddenWindow).After(time.Now()) {
		//dont try to scale hpa min if you scaled it recently , let k8s to cool down the hpa before you make another scaling decision
		return true
	}

	return false
}

func (r *HpaTunerReconciler) UpdateHpaMin(hpaTuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler, newMin int32) (updated bool, err error) {
	r.Log.Info("UpdateHpaMin: ", "newMin", newMin)
	oldMin := *hpa.Spec.MinReplicas

	if oldMin == newMin { //must be some upstream calculation issue
		return false, nil
	}

	hpa.Spec.MinReplicas = &newMin
	if err := r.Client.Update(context.TODO(), hpa); err != nil {
		r.Log.Error(err, "Failed to Update hpa Min", "newMin", newMin)
	}

	if oldMin > newMin {
		hpaTuner.Status.LastDownScaleTime = &metav1.Time{Time: time.Now()}
	} else {
		hpaTuner.Status.LastUpScaleTime = &metav1.Time{Time: time.Now()}
	}
	//hpaTuner.Status.LastUpScaleTime.Time = time.Now() //TODO: put in constructor

	if err := r.Client.Update(context.TODO(), hpaTuner); err != nil {
		r.Log.Error(err, "Failed to Update hpaTuner LastUpScaleTime", "newMin", newMin)
	}

	return true, nil
}

func toString(hpa *scaleV1.HorizontalPodAutoscaler) string {
	var lastScaleTime string

	if hpa.Status.LastScaleTime != nil {
		//lastScaleTime = string(time.Since(hpa.Status.LastScaleTime.Time)) - does not work on 1.5
		lastScaleTime = time.Since(hpa.Status.LastScaleTime.Time).String()
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

func toStringTuner(hpatuner webappv1.HpaTuner) string {

	return fmt.Sprintf("n: %v, status: %v",
		hpatuner.Name,
		hpatuner.Status)
}

func (r *HpaTunerReconciler) scaleToDesired(tuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler) int32 {
	return max(tuner.Spec.MinReplicas, hpa.Status.DesiredReplicas)
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

func canCoolDownHpaMin(tuner *webappv1.HpaTuner, hpa *scaleV1.HorizontalPodAutoscaler, decisionServiceDesired int32) bool {
	if elapsedDownscaleForbiddenWindow(hpa, tuner) {
		//now I can consider letting it cooldown if idle
		if isIdle(hpa) {
			return true
		}
	}

	return false
}

func isIdle(hpa *scaleV1.HorizontalPodAutoscaler) bool {

	//todo, optionally take the idle cpu from hpatunerConfig
	if hpa.Status.CurrentCPUUtilizationPercentage == nil || *hpa.Status.CurrentCPUUtilizationPercentage < 5 {
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
	r.k8sHpaDownScaleTime = time.Minute * 30

	if r.scalingDecisionService == nil { //nil check needed to preserve the stub in testing
		r.scalingDecisionService = CreateScalingDecisionService(r.Log)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.HpaTuner{}).
		Complete(r)
}
