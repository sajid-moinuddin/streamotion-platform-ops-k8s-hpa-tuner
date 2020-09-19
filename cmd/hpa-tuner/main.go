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

package main

import (
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	webappv1 "hpa-tuner/api/v1"
	"hpa-tuner/internal/controllers"
	Flags "hpa-tuner/internal/flags"
	"hpa-tuner/internal/wiring"

	"go.uber.org/zap"
	// +kubebuilder:scaffold:imports
)

const (
	appName   = "hpa-tuner"
	gitCommit = "dirty"
	version   = "devbuild"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = webappv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var cfg wiring.Config

	app := Flags.Flags(appName, gitCommit, version, &cfg)
	if app == nil {
		log.Fatalf("Failed to parse flags")
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Unable to create logger: %s", err.Error())
	}

	logger.Info("Initialising")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		// TODO: Use flags for the ID
		// TODO: Use flags for the port
		LeaderElection:     cfg.EnableLeaderElection,
		LeaderElectionID:   "2ed5900d.streamotion.com.au",
		MetricsBindAddress: cfg.MetricsAddr,
		Port:               9443,
		Scheme:             scheme,
	})

	if err != nil {
		logger.Fatal("unable to start manager: %s", zap.Error(err))
	}

	if err = (&controllers.HpaTunerReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("HpaTuner"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Fatal("unable to create controller", zap.Error(err))
	}
	// +kubebuilder:scaffold:builder

	logger.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Fatal("problem running manager", zap.Error(err))
	}
}
