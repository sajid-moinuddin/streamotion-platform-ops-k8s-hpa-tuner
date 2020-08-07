package controllers

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	//. "github.com/onsi/gomega"
)

var _ = Describe("ScalingDecisionService", func() {
	Context("ScalingDecisionServiceTest", func() {
		It("returns nil if endpoint is not defined in environment", func() {
			decisionService := CreateScalingDecisionService()
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

			decisionService := CreateScalingDecisionService()

			Expect(decisionService).ToNot(BeNil())

			decision, err := decisionService.scalingDecision("test", 1, 2)
			Expect(err).To(BeNil())

			Expect(decision).ToNot(BeNil())
			var expected int32 = int32(99)

			Expect(decision.MinReplicas).To(Equal(expected))
		})

	})
})
