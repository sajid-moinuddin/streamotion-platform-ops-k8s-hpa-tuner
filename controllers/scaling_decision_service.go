package controllers

import (
	"net/http"
	"os"
)

type ScalingDecisionService interface {
	scalingDecision(name string, min int32, current int32) ScalingDecision
}

//// HpaTunerStatus defines the observed state of HpaTuner
//type HpaTunerStatus struct {
//	// Last time a scaleup event was observed
//	LastUpScaleTime *metav1.Time `json:"lastUpScaleTime,omitempty"`
//	// Last time a scale-down event was observed
//	LastDownScaleTime *metav1.Time `json:"lastDownScaleTime,omitempty"`
//}

type ScalingDecision struct {
	MinReplicas int32 `json:"number"`
}

//factory method (TODO: whats the GO style for factory / di?)
func CreateScalingDecisionService() ScalingDecisionService {
	decisionServiceEndPoint, exists := os.LookupEnv("DECISION_SERVICE_ENDPOINT")

	if exists {
		return HttpScalingDecisionService{
			decisionServiceEndpoint: decisionServiceEndPoint,
		}
	} else {
		return nil
	}
}

type HttpScalingDecisionService struct {
	decisionServiceEndpoint string
	Client  *http.Client
}

func (s HttpScalingDecisionService) scalingDecision(name string, min int32, current int32) ScalingDecision {

 	//curl -X GET "http://localhost:8080/api/HorizontalPodAutoscaler?name=hpa-martian-content-qa&current-min=10&current-instance-count=5" -H "accept: application/json"







	return ScalingDecision{
		MinReplicas: 5,
	}
}
