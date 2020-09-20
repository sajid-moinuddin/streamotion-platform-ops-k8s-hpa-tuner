package controllers

import (
	"encoding/json"
	"fmt"
	"hpa-tuner/internal/wiring"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ScalingDecisionService describes a <>. TODO: better comment
type ScalingDecisionService interface {
	scalingDecision(name string, min int32, current int32) (*ScalingDecision, error)
}

// TODO: Remove dead code

//// HpaTunerStatus defines the observed state of HpaTuner
//type HpaTunerStatus struct {
//	// Last time a scaleup event was observed
//	LastUpScaleTime *metav1.Time `json:"lastUpScaleTime,omitempty"`
//	// Last time a scale-down event was observed
//	LastDownScaleTime *metav1.Time `json:"lastDownScaleTime,omitempty"`
//}

// ScalingDecision describes a <> TODO: better comment
type ScalingDecision struct {
	MinReplicas int32 `json:"number"`
}

// CreateScalingDecisionService is a factory method (TODO, SM: whats the GO style for factory / di?)
func CreateScalingDecisionService(logger *zap.Logger, cfg *wiring.Config) ScalingDecisionService {

	if cfg.DecisionServiceEndpoint == "" {
		logger.Info("No decision endpoint supplied")
		return nil
	}

	logger.Info("Using decision service", zap.String("endpoint", cfg.DecisionServiceEndpoint))

	return HTTPScalingDecisionService{
		Client:                  &http.Client{Timeout: time.Second * 10},
		decisionServiceEndpoint: cfg.DecisionServiceEndpoint,
		logger:                  logger,
	}
	return nil
}

// HTTPScalingDecisionService requires a better comment. TODO: Fix that
type HTTPScalingDecisionService struct {
	Client                  *http.Client
	decisionServiceEndpoint string
	logger                  *zap.Logger
}

// DecisionServiceResponse requires a better comment. TODO: Fix that
type DecisionServiceResponse struct {
	Decision struct {
		MinCount int32 `json:"minCount"`
	} `json:"decision"`
}

func (s HTTPScalingDecisionService) scalingDecision(name string, min int32, current int32) (*ScalingDecision, error) {
	s.logger.Info("scalingDecision", zap.String("name", name), zap.Int32("min", min), zap.Int32("current", current))

	//curl -X GET "http://localhost:8080/api/HorizontalPodAutoscaler?name=hpa-martian-content-qa&current-min=10&current-instance-count=5" -H "accept: application/json"
	// TODO Handle error
	req, _ := http.NewRequest("GET", s.decisionServiceEndpoint+"/api/HorizontalPodAutoscaler", nil)

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
		// TODO Should we 'die' (there was a log.Fatel here before)
		s.logger.Fatal("scalingDecision", zap.Any("error", err))
	}

	s.logger.Info("scalingDecision", zap.Any("response", responseData))

	var responseObject DecisionServiceResponse
	json.Unmarshal(responseData, &responseObject)

	s.logger.Info("scalingDecision", zap.Any("--response", responseObject))

	return &ScalingDecision{
		MinReplicas: responseObject.Decision.MinCount,
	}, nil
}
