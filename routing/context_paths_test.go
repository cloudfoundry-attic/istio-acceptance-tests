package routing_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Context Paths", func() {
	var (
		domain            string
		app               string
		hostname          string
		contextPath       string
		helloRoutingAsset = "../assets/hello-golang"
	)

	BeforeEach(func() {
		domain = istioDomain()
		hostname = generator.PrefixedRandomName("IATS", "host")
		contextPath = "/nothing/matters"

		app = generator.PrefixedRandomName("IATS", "APP")
		Expect(cf.Cf("push", app,
			"-p", helloRoutingAsset,
			"-f", fmt.Sprintf("%s/manifest.yml", helloRoutingAsset),
			"-n", hostname,
			"-d", domain,
			"--route-path", contextPath,
			"-i", "1").Wait(defaultTimeout)).To(Exit(0))
	})

	Context("when using a context path", func() {
		It("should route to the appropriate app", func() {
			baseURL := fmt.Sprintf("http://%s.%s", hostname, domain)
			contextPathURL := fmt.Sprintf("http://%s.%s%s", hostname, domain, contextPath)

			Consistently(func() (int, error) {
				return getStatusCode(baseURL)
			}, "15s").Should(Equal(http.StatusNotFound))

			Eventually(func() (int, error) {
				return getStatusCode(contextPathURL)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))

			res, err := http.Get(contextPathURL)
			Expect(err).ToNot(HaveOccurred())

			body, err := ioutil.ReadAll(res.Body)
			Expect(err).ToNot(HaveOccurred())

			type AppResponse struct {
				Greeting string `json:"greeting"`
			}

			var appResponse AppResponse
			err = json.Unmarshal(body, &appResponse)
			Expect(err).ToNot(HaveOccurred())

			Expect(appResponse.Greeting).To(Equal("hello"))
		})
	})

	Context("when unmapping a route with a contextpath", func() {
		It("should receive a 404 during a request", func() {
			contextPathURL := fmt.Sprintf("http://%s.%s%s", hostname, domain, contextPath)

			Eventually(func() (int, error) {
				return getStatusCode(contextPathURL)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))

			Expect(cf.Cf("unmap-route", app, domain,
				"--hostname", hostname,
				"--path", contextPath).Wait(defaultTimeout)).To(Exit(0))

			Eventually(func() (int, error) {
				return getStatusCode(contextPathURL)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusNotFound))
		})
	})
})
