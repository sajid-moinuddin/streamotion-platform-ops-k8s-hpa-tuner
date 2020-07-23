package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"log"

	// "hpa-tuner/controllers"
)

var _ = Describe("HpatunerController", func() {
	logger := log.New(GinkgoWriter, "INFO: ", log.Lshortfile)
	//ctx := context.Background()

	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
	})

	AfterEach(func() {

	})

	Context("test some stuff", func() {

		It("Should Be Able to create HpaTuner CRDs2", func() {
			logger.Println("----------------start test-----------")
			isInteresting := true
			Expect(isInteresting).To(BeTrue())
			Expect(!isInteresting).To(BeFalse())

		})

		It("Should Be Able to create HpaTuner CRDs", func() {
			logger.Println("----------------start test-----------")
			isInteresting := true
			Expect(isInteresting).To(BeTrue())
			Expect(!isInteresting).To(BeFalse())

		})
	})

})
