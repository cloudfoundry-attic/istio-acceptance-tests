package routing_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Weighted Routing", func() {
	var (
		domain              string
		app1                string
		app2                string
		helloRoutingDroplet = "../assets/hello-golang.tgz"
		holaRoutingDroplet  = "../assets/hola-golang.tgz"
	)

	BeforeEach(func() {
		domain = istioDomain()

		app1 = generator.PrefixedRandomName("IATS", "APP1")
		Expect(cf.Cf("push", app1,
			"-s", "cflinuxfs3",
			"-i", "1",
			"-m", "16M",
			"-k", "75M",
			"-d", domain,
			"--hostname", app1,
			"--droplet", helloRoutingDroplet,
			"--no-start").Wait(defaultTimeout)).To(Exit(0))

		app2 = generator.PrefixedRandomName("IATS", "APP2")
		Expect(cf.Cf("push", app2,
			"-s", "cflinuxfs3",
			"-i", "1",
			"-m", "16M",
			"-k", "75M",
			"-d", domain,
			"--hostname", app2,
			"--droplet", holaRoutingDroplet,
			"--no-start").Wait(defaultTimeout)).To(Exit(0))
	})

	Context("when weights are assigned to routes", func() {
		var (
			appGuid1   string
			appGuid2   string
			hostname   string
			routeGuid1 string
			routeURL   string
		)

		BeforeEach(func() {
			hostname = generator.PrefixedRandomName("greetings", "app")
			Expect(cf.Cf("create-route", spaceName(), domain, "--hostname", hostname).Wait(defaultTimeout)).To(Exit(0))

			guid1 := cf.Cf("app", app1, "--guid").Wait(defaultTimeout).Out.Contents()
			appGuid1 = strings.TrimSpace(string(guid1))
			guid2 := cf.Cf("app", app2, "--guid").Wait(defaultTimeout).Out.Contents()
			appGuid2 = strings.TrimSpace(string(guid2))

			routeGuid1 = routeGuid(spaceName(), hostname)
			routeURL = fmt.Sprintf("http://%s.%s", hostname, domain)
		})

		It("balances routes according to the weights assigned to them", func() {
			mapWeightedRoute(routeGuid1, appGuid1, 2)
			mapWeightedRoute(routeGuid1, appGuid2, 2)

			Expect(cf.Cf("start", app1).Wait(defaultTimeout)).To(Exit(0))
			Expect(cf.Cf("start", app2).Wait(defaultTimeout)).To(Exit(0))

			// Make sure both apps are individually routable
			// before checking the shared weighted route
			unweightedRouteUrl1 := fmt.Sprintf("http://%s.%s", app1, domain)
			Eventually(func() (int, error) {
				return getStatusCode(unweightedRouteUrl1)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
			unweightedRouteUrl2 := fmt.Sprintf("http://%s.%s", app2, domain)
			Eventually(func() (int, error) {
				return getStatusCode(unweightedRouteUrl2)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))

			res, err := http.Get(routeURL)
			Expect(err).ToNot(HaveOccurred())
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			Expect(err).ToNot(HaveOccurred())

			type AppResponse struct {
				Greeting string `json:"greeting"`
			}

			var app1Resp AppResponse
			err = json.Unmarshal(body, &app1Resp)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() (AppResponse, error) {
				res, err := http.Get(routeURL)
				if err != nil {
					return AppResponse{}, err
				}

				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					return AppResponse{}, err
				}

				var app2Resp AppResponse
				err = json.Unmarshal(body, &app2Resp)
				if err != nil {
					return AppResponse{}, err
				}

				return app2Resp, nil
			}, defaultTimeout, time.Second).ShouldNot(Equal(app1Resp))
		})
	})
})

func mapWeightedRoute(routeGuid, appGuid string, weight int) {
	Expect(cf.Cf(
		"curl",
		"/v3/route_mappings",
		"-H", "Content-type: application/json",
		"-X", "POST",
		"-d", fmt.Sprintf(`{
					"relationships": {
						"app": { "guid": "%s" },
						"route": { "guid": "%s" }
					},
					"weight": %d
				}`, appGuid, routeGuid, weight),
	).Wait(defaultTimeout)).To(Exit(0))
}
