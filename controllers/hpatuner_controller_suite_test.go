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
	"github.com/onsi/gomega/gexec"
	"k8s.io/client-go/kubernetes"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	webappv1 "hpa-tuner/api/v1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
// VERY USEFUL:
//https://itnext.io/taking-a-kubernetes-operator-to-production-bc59708db420
//https://itnext.io/testing-kubernetes-operators-with-ginkgo-gomega-and-the-operator-runtime-6ad4c2492379

var cfg *rest.Config
var k8sClient client.Client
var clientSet *kubernetes.Clientset
var k8sManager ctrl.Manager

var testEnv *envtest.Environment

var fakeDecisionService FakeScalingDecisionService

type FakeScalingDecisionService struct {
	FakeDecision *ScalingDecision
}

func (s FakeScalingDecisionService) scalingDecision(name string, min int32, current int32) (*ScalingDecision,error) {
	//println(fmt.Printf("-------------object ref: %v" , s.FakeDecision))
	return s.FakeDecision, nil
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logger := zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))

	logf.SetLogger(logger)

	useCluster := true

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		UseExistingCluster:       &useCluster,
		AttachControlPlaneOutput: true,
		CRDDirectoryPaths:        []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = webappv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		Port:               9999,
		MetricsBindAddress: "0",
	})

	Expect(err).ToNot(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	clientSet, err = kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred())

	controllerLog := ctrl.Log.WithName("controllers").WithName("Run")

	fakeDecisionService = FakeScalingDecisionService{
		FakeDecision: &ScalingDecision{MinReplicas: 7},
	}

	//register the controller to controller manager
	err = (&HpaTunerReconciler{
		Client:                 k8sClient,
		Log:                    controllerLog,
		Scheme:                 nil,
		eventRecorder:          k8sManager.GetEventRecorderFor("hpa-tuner"),
		clientSet:              clientSet,
		syncPeriod:             0,
		scalingDecisionService: fakeDecisionService,
	}).SetupWithManager(k8sManager)

	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	close(done)
}, 120)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	gexec.KillAndWait(5 * time.Second)

	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
