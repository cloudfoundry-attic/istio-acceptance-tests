package routing_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Routing", func() {
	var (
		domain              string
		app                 string
		helloRoutingDroplet = "../assets/hello-golang.tgz"
		appURL              string
	)

	BeforeEach(func() {
		domain = istioDomain()

		app = generator.PrefixedRandomName("IATS", "APP")
		Expect(cf.Cf("push", app,
			"-d", domain,
			"-s", "cflinuxfs3",
			"--droplet", helloRoutingDroplet,
			"-i", "1",
			"-m", "16M",
			"-k", "75M").Wait(defaultTimeout)).To(Exit(0))
		appURL = fmt.Sprintf("http://%s.%s", app, domain)

		Eventually(func() (int, error) {
			return getStatusCode(appURL)
		}, defaultTimeout).Should(Equal(http.StatusOK))
	})

	Context("when an app is pushed to the istio domain with frontend certs", func() {
		BeforeEach(func() {
			if Config.WildcardCa == "" {
				Skip("skipping tls termination test, no wildcard ca supplied")
			}
		})

		It("response to HTTPS requests", func() {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM([]byte(Config.WildcardCa))

			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: caCertPool,
					},
				},
			}
			// Frontend wildcard certs are setup in the manifest for the env
			httpsAppURL := fmt.Sprintf("https://%s.%s", app, domain)

			Eventually(func() (int, error) {
				res, err := client.Get(httpsAppURL)
				res.Body.Close()
				return res.StatusCode, err
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
		})
	})

	Context("when the app is stopped", func() {
		It("returns a 503", func() {
			Expect(cf.Cf("stop", app).Wait(defaultTimeout)).To(Exit(0))

			Eventually(func() (int, error) {
				return getStatusCode(appURL)
			}, defaultTimeout).Should(Equal(http.StatusServiceUnavailable))
		})
	})

	Context("when an app has many routes", func() {
		var (
			hostnameOne string
			hostnameTwo string
		)

		BeforeEach(func() {
			hostnameOne = generator.PrefixedRandomName("IATS", "host")
			hostnameTwo = hostnameOne + "-2"

			mapRouteOneCmd := cf.Cf("map-route", app, domain, "--hostname", hostnameOne)
			Expect(mapRouteOneCmd.Wait(defaultTimeout)).To(Exit(0))
			mapRouteTwoCmd := cf.Cf("map-route", app, domain, "--hostname", hostnameTwo)
			Expect(mapRouteTwoCmd.Wait(defaultTimeout)).To(Exit(0))
		})

		It("requests succeed to all routes", func() {
			Eventually(func() (int, error) {
				appURLOne := fmt.Sprintf("http://%s.%s", hostnameOne, domain)
				return getStatusCode(appURLOne)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))

			Eventually(func() (int, error) {
				appURLTwo := fmt.Sprintf("http://%s.%s", hostnameTwo, domain)
				return getStatusCode(appURLTwo)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
		})

		It("successfully unmaps routes and request continue to succeed for mapped routes", func() {
			unmapRouteOneCmd := cf.Cf("unmap-route", app, domain, "--hostname", hostnameOne)
			Expect(unmapRouteOneCmd.Wait(defaultTimeout)).To(Exit(0))

			Eventually(func() (int, error) {
				appURLOne := fmt.Sprintf("http://%s.%s", hostnameOne, domain)
				return getStatusCode(appURLOne)
			}, defaultTimeout).Should(Equal(http.StatusNotFound))

			Eventually(func() (int, error) {
				appURLTwo := fmt.Sprintf("http://%s.%s", hostnameTwo, domain)
				return getStatusCode(appURLTwo)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
		})
	})

	Context("when an app has a user-provided internal route", func() {
		It("requests are not externally accessible to the internal route", func() {
			timeout := time.Duration(time.Second * 10)
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr, Timeout: timeout}
			hostname := generator.PrefixedRandomName("IATS", "HOST")
			mapRouteInternalCmd := cf.Cf("map-route", app, internalDomain(), "--hostname", hostname)
			Expect(mapRouteInternalCmd.Wait(defaultTimeout)).To(Exit(0))

			req, err := http.NewRequest("GET", fmt.Sprintf("http://envoy.%s", domain), nil)
			Expect(err).NotTo(HaveOccurred())
			req.Host = fmt.Sprintf("%s.%s", hostname, internalDomain())

			Eventually(func() (int, error) {
				resp, err := client.Do(req)
				if err != nil {
					return 0, err
				}

				return resp.StatusCode, err
			}, defaultTimeout, time.Second).Should(Equal(http.StatusNotFound))
		})
	})

	Context("route mappings", func() {
		Context("mapping a route using both CAPI endpoints", func() {
			var (
				appGuid  string
				hostname string
			)

			BeforeEach(func() {
				appGuid = applicationGuid(app)
				hostname = generator.PrefixedRandomName("iats", "host")
				Expect(cf.Cf("create-route", spaceName(), domain, "--hostname", hostname).Wait(defaultTimeout)).To(Exit(0))
			})

			It("can map route using Apps API", func() {
				routeGuid := routeGuid(spaceName(), hostname)
				Expect(cf.Cf("curl", fmt.Sprintf("v2/apps/%s/routes/%s", appGuid, routeGuid), "-X", "PUT").Wait(defaultTimeout)).To(Exit(0))

				Eventually(func() (int, error) {
					appURL := fmt.Sprintf("http://%s.%s", hostname, domain)
					return getStatusCode(appURL)
				}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
			})

			It("can map route using Routes API", func() {
				routeGuid := routeGuid(spaceName(), hostname)
				Expect(cf.Cf("curl", fmt.Sprintf("v2/routes/%s/apps/%s", routeGuid, appGuid), "-X", "PUT").Wait(defaultTimeout)).To(Exit(0))

				Eventually(func() (int, error) {
					appURL := fmt.Sprintf("http://%s.%s", hostname, domain)
					return getStatusCode(appURL)
				}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
			})
		})
	})
})

type Instance struct {
	Index string `json:"instance_index"`
	GUID  string `json:"instance_guid"`
}
