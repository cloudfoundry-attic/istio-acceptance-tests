package routing_test

import (
	"fmt"
	"net/http"

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
		domain            string
	)

	BeforeEach(func() {
		domain = IstioDomain()

		app = random_name.CATSRandomName("APP")
		pushCmd := cf.Cf("push", app,
			"-p", helloRoutingAsset,
			"-f", fmt.Sprintf("%s/manifest.yml", helloRoutingAsset),
			"-d", domain).Wait(defaultTimeout)
		Expect(pushCmd).To(Exit(0))

		res, err := http.Get(fmt.Sprintf("http://%s.%s", app, domain))
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

			res, err := http.Get(fmt.Sprintf("http://%s.%s", app, domain))
			Expect(err).ToNot(HaveOccurred())
			Expect(res.StatusCode).To(Equal(503))
		})
	})
})
