package handlers_test

import (
	"net/http"
	"net/http/httptest"

	"io/ioutil"

	"code.cloudfoundry.org/istio-acceptance-tests/assets/proxy/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EchoSourceIPHandler", func() {
	var (
		handler *handlers.EchoSourceIPHandler
		resp    *httptest.ResponseRecorder
		req     *http.Request
	)

	BeforeEach(func() {
		handler = &handlers.EchoSourceIPHandler{}
		resp = httptest.NewRecorder()
	})

	Describe("GET", func() {
		BeforeEach(func() {
			var err error
			req, err = http.NewRequest("GET", "/echosourceip", nil)
			req.RemoteAddr = "foo"
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns a body with the source ip", func() {
			handler.ServeHTTP(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))
			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal("foo"))
		})
	})
})
