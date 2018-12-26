package routing_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Automatic Retries: Internal Routes", func() {
	var (
		domain               string
		internalDomain       string
		proxy                string
		flakyBackend         string
		proxyDroplet         = "../assets/proxy.tgz"
		flakyBackendApp      = "../assets/flaky-backend"
		flakyBackendManifest = "../assets/flaky-backend/manifest.yml"

		internalRoute string
		routeURL      string
	)

	BeforeEach(func() {
		if !Config.GetIncludeInternalRouteTests() {
			Skip("'include_internal_route_tests' is false in config. Skipping automatic retries test.")
		}

		domain = systemDomain()
		internalDomain = internalIstioDomain()

		proxy = generator.PrefixedRandomName("IATS", "APP1")
		Expect(cf.Cf("push", proxy,
			"-d", domain,
			"-s", "cflinuxfs3",
			"--hostname", proxy,
			"--droplet", proxyDroplet,
			"-i", "1",
			"-m", "32M",
			"-k", "75M").Wait(defaultTimeout)).To(Exit(0))

		flakyBackend = generator.PrefixedRandomName("IATS", "APP2")
		Expect(cf.Cf("push", flakyBackend,
			"-s", "cflinuxfs3",
			"-d", internalDomain,
			"--hostname", flakyBackend,
			"-f", flakyBackendManifest,
			"-p", flakyBackendApp).Wait(defaultTimeout)).To(Exit(0))

		Expect(cf.Cf("add-network-policy",
			proxy, "--destination-app", flakyBackend).Wait(defaultTimeout)).To(Exit(0))

		internalRoute = fmt.Sprintf("%s.%s:8080", flakyBackend, internalDomain)
		routeURL = fmt.Sprintf("http://%s.%s/proxy/%s", proxy, domain, internalRoute)

		By("waiting for the app to start and become reachable")
		Eventually(func() (int, error) {
			return getStatusCode(routeURL)
		}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
	})

	It("automatically retries for the client if a request fails", func() {
		By("ensuring the request succeeds multiple times with a flaky backend")
		res, err := http.Get(routeURL)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))

		res, err = http.Get(routeURL)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))

		res, err = http.Get(routeURL)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
	})
})
