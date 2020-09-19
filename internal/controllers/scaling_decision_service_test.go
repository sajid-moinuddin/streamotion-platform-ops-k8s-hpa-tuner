package controllers

import (
	"fmt"
	"hpa-tuner/internal/wiring"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	//. "github.com/onsi/gomega"
)

var _ = Describe("ScalingDecisionService", func() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Unable to create logger: %s", err.Error())
	}

	var cfg wiring.Config

	Context("ScalingDecisionServiceTest", func() {
		It("returns nil if endpoint is not defined in environment", func() {

			decisionService := CreateScalingDecisionService(logger, &cfg)
			Expect(decisionService).To(BeNil())
		})

		It("test get mincount", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, ` {"decision":{"minCount":99}}`)
			}))
			defer ts.Close()

			log.Printf("----------%v", ts.URL)

			os.Setenv("DECISION_SERVICE_ENDPOINT", ts.URL+"/api/HorizontalPodAutoscaler?name=hpa-martian-content-qa")

			//os.Setenv("DECISION_SERVICE_ENDPOINT", "http://localhost:8080")

			decisionService := CreateScalingDecisionService(logger, &cfg)

			Expect(decisionService).ToNot(BeNil())

			decision, err := decisionService.scalingDecision("test", 1, 2)
			Expect(err).To(BeNil())

			Expect(decision).ToNot(BeNil())
			var expected int32 = int32(99)

			Expect(decision.MinReplicas).To(Equal(expected))
		})

	})
})
