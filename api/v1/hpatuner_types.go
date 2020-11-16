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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HpaTunerSpec defines the desired state of HpaTuner
type HpaTunerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=6000
	DownscaleForbiddenWindowSeconds int32 `json:"downscaleForbiddenWindowSeconds,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=600
	UpscaleForbiddenWindowAfterDownScaleSeconds int32 `json:"upscaleForbiddenWindowAfterDownscaleSeconds,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	ScaleUpLimitFactor int32 `json:"scaleUpLimitFactor,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=20
	ScaleUpLimitMinimum int32 `json:"scaleUpLimitMinimum,omitempty"`
	// // +kubebuilder:validation:Minimum=0.01
	// // +kubebuilder:validation:Maximum=0.99
	// Tolerance float64 `json:"tolerance,omitempty"`
	// part of HorizontalPodAutoscalerSpec, see comments in the k8s-1.10.8 repo: staging/src/k8s.io/api/autoscaling/v1/types.go
	// reference to scaled resource; horizontal pod autoscaler will learn the current resource consumption
	// and will set the desired number of pods by using its Scale subresource.
	ScaleTargetRef CrossVersionObjectReference `json:"scaleTargetRef"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	MinReplicas int32 `json:"minReplicas,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	MaxReplicas int32 `json:"maxReplicas"`

	// +kubebuilder:default := false
	UseDecisionService bool `json:"useDecisionService"`
}

// CrossVersionObjectReference contains enough information to let you identify the referred resource.
type CrossVersionObjectReference struct {
	// Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds"
	Kind string `json:"kind"`
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// HpaTunerStatus defines the observed state of HpaTuner
type HpaTunerStatus struct {
	// Last time I upped the hpaMin
	LastUpScaleTime *metav1.Time `json:"lastUpScaleTime,omitempty"`

	// Last time I downed the hpaMin
	LastDownScaleTime *metav1.Time `json:"lastDownScaleTime,omitempty"`
}

// +kubebuilder:object:root=true

// HpaTuner is the Schema for the hpatuners API
type HpaTuner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HpaTunerSpec   `json:"spec,omitempty"`
	Status HpaTunerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HpaTunerList contains a list of HpaTuner
type HpaTunerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HpaTuner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HpaTuner{}, &HpaTunerList{})
}
