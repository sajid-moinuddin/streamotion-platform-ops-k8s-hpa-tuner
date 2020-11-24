package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type ScalingDecisionService interface {
	scalingDecision(name string, min int32, current int32) (*ScalingDecision, error)
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
func CreateScalingDecisionService(log logr.Logger) ScalingDecisionService {
	decisionServiceEndPoint, exists := os.LookupEnv("DECISION_SERVICE_ENDPOINT")

	if exists {
		log.Info("USING", "ScalingDecisionService", decisionServiceEndPoint)
		return HttpScalingDecisionService{
			decisionServiceEndpoint: decisionServiceEndPoint,
			Client: &http.Client{
				Timeout: time.Second * 10,
			},
			log: log,
		}
	} else {
		log.Info("***** NO Environment var called: DECISION_SERVICE_ENDPOINT *******")
		return nil
	}
}

type HttpScalingDecisionService struct {
	decisionServiceEndpoint string
	Client                  *http.Client
	log                     logr.Logger
}

type DecisionServiceResponse struct {
	Decision struct {
		MinCount int32 `json:"minCount"`
	} `json:"decision"`
}

//TODO: the following works but verify this is the best way to do rest calls in GO
func (s HttpScalingDecisionService) scalingDecision(name string, min int32, current int32) (*ScalingDecision, error) {
	log := s.log.WithValues("name", name)
	log.V(5).Info("get scalingDecision", "name", name, "min", min, "current", current)

	//curl -X GET "http://localhost:8080/api/HorizontalPodAutoscaler?name=hpa-martian-content-qa&current-min=10&current-instance-count=5" -H "accept: application/json"
	req, _ := http.NewRequest("GET", s.decisionServiceEndpoint+"/api/HorizontalPodAutoscaler", nil)

	q := req.URL.Query()
	q.Add("name", name)
	q.Add("current-min", fmt.Sprint(min))
	q.Add("current-instance-count", fmt.Sprint(current))

	req.URL.RawQuery = q.Encode()
	req.Header.Add("Content-Type", "application/json")
	log.V(5).Info("Encoded", "url", req.URL.RawQuery)

	response, err := s.Client.Do(req)

	if err != nil {
		log.Error(err, "failed to get decision from decision service")
		return nil, err
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error(err, "Failed getting decision service resp")
		return nil, err
	}

	log.V(5).Info("Decision Service response", "resp", string(responseData))

	var responseObject DecisionServiceResponse
	json.Unmarshal(responseData, &responseObject)

	return &ScalingDecision{
		MinReplicas: responseObject.Decision.MinCount,
	}, nil
}
