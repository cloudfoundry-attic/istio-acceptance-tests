package routing_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	routing_helpers "code.cloudfoundry.org/cf-routing-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Routing", func() {
	var (
		app               string
		helloRoutingAsset = "../assets/golang"
		appURL            string
	)

	BeforeEach(func() {
		domain := IstioDomain()

		app = random_name.CATSRandomName("APP")
		pushCmd := cf.Cf("push", app,
			"-p", helloRoutingAsset,
			"-f", fmt.Sprintf("%s/manifest.yml", helloRoutingAsset),
			"-d", domain,
			"-i", "2").Wait(defaultTimeout)
		Expect(pushCmd).To(Exit(0))
		appURL = fmt.Sprintf("http://%s.%s", app, domain)

		res, err := http.Get(appURL)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(200))
	})

	AfterEach(func() {
		routing_helpers.AppReport(app, defaultTimeout)
		routing_helpers.DeleteApp(app, defaultTimeout)
	})

	Context("when the app is stopped", func() {
		It("returns a 503", func() {
			stopCmd := cf.Cf("stop", app).Wait(defaultTimeout)
			Expect(stopCmd).To(Exit(0))

			Eventually(func() (int, error) {
				res, err := http.Get(appURL)
				if err != nil {
					return 0, err
				}
				return res.StatusCode, err
			}, defaultTimeout).Should(Equal(503))
		})
	})

	Context("when the app has many instances", func() {
		It("routes in a round robin", func() {
			res, err := http.Get(appURL)
			Expect(err).ToNot(HaveOccurred())

			body, err := ioutil.ReadAll(res.Body)
			Expect(err).ToNot(HaveOccurred())

			type Instance struct {
				Index string `json:"instance_index"`
				GUID  string `json:"instance_guid"`
			}
			var instanceOne Instance
			err = json.Unmarshal(body, &instanceOne)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() (Instance, error) {
				res, err := http.Get(appURL)
				if err != nil {
					return Instance{}, err
				}

				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					return Instance{}, err
				}

				var instanceTwo Instance
				err = json.Unmarshal(body, &instanceTwo)
				if err != nil {
					return Instance{}, err
				}
				return instanceTwo, nil
			}, defaultTimeout, time.Second).ShouldNot(Equal(instanceOne))
		})
	})
})
