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
		internalDomain      string
		app1                string
		app2                string
		proxyFrontend       string
		proxyDroplet        = "../assets/proxy.tgz"
		helloRoutingDroplet = "../assets/hello-golang.tgz"
		holaRoutingDroplet  = "../assets/hola-golang.tgz"
	)

	BeforeEach(func() {
		domain = istioDomain()
		internalDomain = internalIstioDomain()

		proxyFrontend = generator.PrefixedRandomName("iats", "proxy1")
		Expect(cf.Cf("push", proxyFrontend,
			"-s", "cflinuxfs3",
			"-i", "1",
			"-m", "16M",
			"-k", "75M",
			"-d", domain,
			"--hostname", proxyFrontend,
			"--droplet", proxyDroplet).Wait(defaultTimeout)).To(Exit(0))

		app1 = generator.PrefixedRandomName("iats", "app1")
		Expect(cf.Cf("push", app1,
			"-s", "cflinuxfs3",
			"-i", "1",
			"-m", "16M",
			"-k", "75M",
			"-d", domain,
			"--hostname", app1,
			"--droplet", helloRoutingDroplet,
			"--no-start").Wait(defaultTimeout)).To(Exit(0))
		Expect(cf.Cf("map-route", app1, internalDomain, "--hostname", app1).Wait(defaultTimeout)).To(Exit(0))

		app2 = generator.PrefixedRandomName("iats", "app2")
		Expect(cf.Cf("push", app2,
			"-s", "cflinuxfs3",
			"-i", "1",
			"-m", "16M",
			"-k", "75M",
			"-d", domain,
			"--hostname", app2,
			"--droplet", holaRoutingDroplet,
			"--no-start").Wait(defaultTimeout)).To(Exit(0))
		Expect(cf.Cf("map-route", app2, internalDomain, "--hostname", app2).Wait(defaultTimeout)).To(Exit(0))

		Expect(cf.Cf("add-network-policy", proxyFrontend, "--destination-app", app1).Wait(defaultTimeout)).To(Exit(0))
		Expect(cf.Cf("add-network-policy", proxyFrontend, "--destination-app", app2).Wait(defaultTimeout)).To(Exit(0))
	})

	Context("when weights are assigned to routes", func() {
		var (
			appGuid1                string
			appGuid2                string
			externalHostname        string
			internalHostname        string
			externalRouteGuid       string
			internalRouteGuid       string
			externalRouteURL        string
			proxiedInternalRouteURL string
		)

		BeforeEach(func() {
			externalHostname = generator.PrefixedRandomName("greetings", "app")
			Expect(cf.Cf("create-route", spaceName(), domain, "--hostname", externalHostname).Wait(defaultTimeout)).To(Exit(0))

			internalHostname = generator.PrefixedRandomName("greetings", "app")
			Expect(cf.Cf("create-route", spaceName(), internalDomain, "--hostname", internalHostname).Wait(defaultTimeout)).To(Exit(0))

			guid1 := cf.Cf("app", app1, "--guid").Wait(defaultTimeout).Out.Contents()
			appGuid1 = strings.TrimSpace(string(guid1))
			guid2 := cf.Cf("app", app2, "--guid").Wait(defaultTimeout).Out.Contents()
			appGuid2 = strings.TrimSpace(string(guid2))

			externalRouteGuid = routeGuid(spaceName(), externalHostname)
			externalRouteURL = fmt.Sprintf("http://%s.%s", externalHostname, domain)

			internalRouteGuid = routeGuid(spaceName(), internalHostname)
			proxiedInternalRouteURL = fmt.Sprintf("http://%s.%s/proxy/%s.%s:8080", proxyFrontend, domain, internalHostname, internalDomain)
		})

		It("balances internal routes according to the weights assigned to them", func() {
			mapWeightedRoute(internalRouteGuid, appGuid1, 1)
			mapWeightedRoute(internalRouteGuid, appGuid2, 9)

			Expect(cf.Cf("start", app1).Wait(defaultTimeout)).To(Exit(0))
			Expect(cf.Cf("start", app2).Wait(defaultTimeout)).To(Exit(0))

			// Make sure both apps are individually routable
			// before checking the shared weighted route
			isUpAndRoutable(fmt.Sprintf("http://%s.%s/proxy/%s.%s:8080", proxyFrontend, domain, app1, internalDomain))
			isUpAndRoutable(fmt.Sprintf("http://%s.%s/proxy/%s.%s:8080", proxyFrontend, domain, app2, internalDomain))

			time.Sleep(60 * time.Second)
			var app1RespCount, app2RespCount int
			for i := 0; i < 100; i++ {
				switch greetingFromApp(proxiedInternalRouteURL) {
				case "hello":
					app1RespCount++
				case "hola":
					app2RespCount++
				}
			}

			Expect(app1RespCount).To(BeNumerically("~", 10, 10), `given a 10% route weight for app 1, the expected response count is ~10`)
			Expect(app2RespCount).To(BeNumerically("~", 90, 10), `given a 90% route weight for app 2, the expected response count is ~90`)
		})

		It("balances external routes according to the weights assigned to them", func() {
			mapWeightedRoute(externalRouteGuid, appGuid1, 1)
			mapWeightedRoute(externalRouteGuid, appGuid2, 9)

			Expect(cf.Cf("start", app1).Wait(defaultTimeout)).To(Exit(0))
			Expect(cf.Cf("start", app2).Wait(defaultTimeout)).To(Exit(0))

			// Make sure both apps are individually routable
			// before checking the shared weighted route
			isUpAndRoutable(fmt.Sprintf("http://%s.%s", app1, domain))
			isUpAndRoutable(fmt.Sprintf("http://%s.%s", app2, domain))

			var app1RespCount, app2RespCount int
			for i := 0; i < 100; i++ {
				switch greetingFromApp(externalRouteURL) {
				case "hello":
					app1RespCount++
				case "hola":
					app2RespCount++
				}
			}

			Expect(app1RespCount).To(BeNumerically("~", 10, 10), `given a 10% route weight for app 1, the expected response count is ~10`)
			Expect(app2RespCount).To(BeNumerically("~", 90, 10), `given a 90% route weight for app 2, the expected response count is ~90`)
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

func isUpAndRoutable(route string) {
	Eventually(func() (int, error) {
		return getStatusCode(route)
	}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
}

func greetingFromApp(route string) string {
	res, err := http.Get(route)
	Expect(err).ToNot(HaveOccurred())
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	Expect(err).ToNot(HaveOccurred())

	type AppResponse struct {
		Greeting string `json:"greeting"`
	}

	var appResp AppResponse
	err = json.Unmarshal(body, &appResp)
	Expect(err).ToNot(HaveOccurred())

	return appResp.Greeting
}
