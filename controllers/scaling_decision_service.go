package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
func CreateScalingDecisionService() ScalingDecisionService {
	log.Printf("****** Creating Decision Service!! ")
	decisionServiceEndPoint, exists := os.LookupEnv("DECISION_SERVICE_ENDPOINT")

	if exists {
		return HttpScalingDecisionService{
			decisionServiceEndpoint: decisionServiceEndPoint,
			Client: &http.Client{
				Timeout: time.Second * 10,
			},
		}
	} else {
		return nil
	}
}

type HttpScalingDecisionService struct {
	decisionServiceEndpoint string
	Client  *http.Client
}


type DecisionServiceResponse struct {
	Decision struct {
		MinCount int32 `json:"minCount"`
	} `json:"decision"`
}


func (s HttpScalingDecisionService) scalingDecision(name string, min int32, current int32) (*ScalingDecision, error) {
	log.Printf("name %v , min: %v, current: %v", name, min, current)

 	//curl -X GET "http://localhost:8080/api/HorizontalPodAutoscaler?name=hpa-martian-content-qa&current-min=10&current-instance-count=5" -H "accept: application/json"
	req, _ := http.NewRequest("GET", s.decisionServiceEndpoint + "/api/HorizontalPodAutoscaler", nil)

	q := req.URL.Query()
	q.Add("name", name)
	q.Add("current-min", fmt.Sprint(min))
	q.Add("current-instance-count", fmt.Sprint(current))

	req.URL.RawQuery = q.Encode()
	req.Header.Add("Content-Type", "application/json")
	fmt.Printf("Encoded URL is %q\n", req.URL.RawQuery)

	response, err := s.Client.Do(req)



	if err != nil {
		log.Fatal(err)
		return nil, err
	}


	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	log.Printf("resp: %v", string(responseData))


	var responseObject DecisionServiceResponse
    json.Unmarshal(responseData, &responseObject)

	log.Printf("--response: %v", responseObject)

	return &ScalingDecision{
		MinReplicas: responseObject.Decision.MinCount,
	}, nil
}

