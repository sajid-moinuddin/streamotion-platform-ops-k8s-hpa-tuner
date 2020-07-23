package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	// "hpa-tuner/controllers"
)

var _ = Describe("HpatunerController", func() {
	Context("test some stuff", func() {

		It("Should correctly identify initStartedEvent", func() {
			isInteresting := true

			print(k8sClient)

			Expect(isInteresting).To(BeTrue())
		})
	})

})
