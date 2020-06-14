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
	"log"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	webappv1 "hpa-tuner/api/v1"
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
	Log    logr.Logger
	Scheme *runtime.Scheme

	syncPeriod time.Duration
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
	log.Printf("##: fetched %v \n", hpaTuner)

	hpaRef := hpaTuner.Spec.ScaleTargetRef

	log.Printf("hparef: %v \n", hpaRef)

	//TODO: check validity of hpaTuner

	// kind := chpa.Spec.ScaleTargetRef.Kind
	hpaNamespace := hpaTuner.Namespace
	hpaName := hpaTuner.Spec.ScaleTargetRef.Name
	hpaNamespacedName := types.NamespacedName{Namespace: hpaNamespace, Name: hpaName}

	hpa := &v1.HorizontalPodAutoscaler{}
	if err := r.Get(ctx, hpaNamespacedName, hpa); err != nil {
		// Error reading the object, repeat later
		rlog.Error(err, "Error reading HPA: ", "hpa", hpaNamespacedName)
		return resRepeat, nil
	}

	log.Printf("*Read HPA: %v", hpa)

	// -----------------------------------------------------------------------------------
	log.Printf("") // to have clear separation between previous and current reconcile run
	log.Printf("--end: ----------------------------------------------------------------------------------------------------")

	// resRepeat will be returned if we want to re-run reconcile process
	// NB: we can't return non-nil err, as the "reconcile" msg will be added to the rate-limited queue
	// so that it'll slow down if we have several problems in a row

	return resRepeat, nil
}

func (r *HpaTunerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	r.syncPeriod = defaultSyncPeriod

	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.HpaTuner{}).
		Complete(r)
}
