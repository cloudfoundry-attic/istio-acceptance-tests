package routing_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Routing", func() {
	var (
		domain            string
		app               string
		helloRoutingAsset = "../assets/hello-golang"
		appURL            string
	)

	BeforeEach(func() {
		domain = istioDomain()

		app = generator.PrefixedRandomName("IATS", "APP")
		Expect(cf.Cf("push", app,
			"-p", helloRoutingAsset,
			"-f", fmt.Sprintf("%s/manifest.yml", helloRoutingAsset),
			"-d", domain,
			"-i", "1").Wait(defaultTimeout)).To(Exit(0))
		appURL = fmt.Sprintf("http://%s.%s", app, domain)

		Eventually(func() (int, error) {
			return getStatusCode(appURL)
		}, defaultTimeout).Should(Equal(http.StatusOK))
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
			hostnameOne = generator.PrefixedRandomName("IATS", "HOST")
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

			req, err := http.NewRequest("GET", fmt.Sprintf("http://envoy.%s", istioDomain()), nil)
			Expect(err).NotTo(HaveOccurred())
			req.Host = fmt.Sprintf("%s.%s", hostname, internalDomain())

			Eventually(func() (int, error) {
				resp, err := client.Do(req)
				return resp.StatusCode, err
			}, defaultTimeout).Should(Equal(http.StatusNotFound))
		})
	})

	Context("route mappings", func() {
		var (
			space string
			org   string
		)

		BeforeEach(func() {
			space = TestSetup.TestSpace.SpaceName()
			org = TestSetup.TestSpace.OrganizationName()
		})

		It("can map a route with a private domain", func() {
			var privateHostname string
			privateDomain := fmt.Sprintf("%s.%s", generator.PrefixedRandomName("iats", "private"), domain)

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), defaultTimeout, func() {
				Expect(cf.Cf("create-domain", org, privateDomain).Wait(defaultTimeout)).To(Exit(0))
			})

			privateHostname = fmt.Sprintf("someApp-%d", time.Now().UnixNano)
			Expect(cf.Cf("map-route", app, privateDomain, "--hostname", privateHostname).Wait(defaultTimeout)).To(Exit(0))

			Eventually(func() (int, error) {
				appURL := fmt.Sprintf("http://%s.%s", privateHostname, privateDomain)
				return getStatusCode(appURL)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
		})

		Context("mapping a route using both CAPI endpoints", func() {
			var (
				aGuid    string
				hostname string
			)

			BeforeEach(func() {
				aGuid = appGuid(app)
				hostname = generator.PrefixedRandomName("iats", "host")
				Expect(cf.Cf("create-route", space, domain, "--hostname", hostname).Wait(defaultTimeout)).To(Exit(0))
			})

			It("can map route using Apps API", func() {
				routeGuid := routeGuid(space, hostname)
				Expect(cf.Cf("curl", fmt.Sprintf("v2/apps/%s/routes/%s", aGuid, routeGuid), "-X", "PUT").Wait(defaultTimeout)).To(Exit(0))

				Eventually(func() (int, error) {
					appURL := fmt.Sprintf("http://%s.%s", hostname, domain)
					return getStatusCode(appURL)
				}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
			})

			It("can map route using Routes API", func() {
				routeGuid := routeGuid(space, hostname)
				Expect(cf.Cf("curl", fmt.Sprintf("v2/routes/%s/apps/%s", routeGuid, aGuid), "-X", "PUT").Wait(defaultTimeout)).To(Exit(0))

				Eventually(func() (int, error) {
					appURL := fmt.Sprintf("http://%s.%s", hostname, domain)
					return getStatusCode(appURL)
				}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
			})
		})
	})

	Context("round robin", func() {
		Context("when the app has many instances", func() {
			BeforeEach(func() {
				Expect(cf.Cf("scale", app, "-i", "2").Wait(defaultTimeout)).To(Exit(0))
			})

			It("successfully load balances between instances", func() {
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

		Context("when mapping a route to multiple apps", func() {
			var (
				holaRoutingAsset = "../assets/hola-golang"
				appTwo           string
				appTwoURL        string
				hostname         string
			)

			BeforeEach(func() {
				domain = istioDomain()

				appTwo = generator.PrefixedRandomName("IATS", "APP")
				Expect(cf.Cf("push", appTwo,
					"-p", holaRoutingAsset,
					"-f", fmt.Sprintf("%s/manifest.yml", holaRoutingAsset),
					"-d", domain,
					"-i", "1").Wait(defaultTimeout)).To(Exit(0))
				appTwoURL = fmt.Sprintf("http://%s.%s", appTwo, domain)

				Eventually(func() (int, error) {
					return getStatusCode(appTwoURL)
				}, defaultTimeout).Should(Equal(http.StatusOK))

				space := TestSetup.TestSpace.SpaceName()
				hostname = "greetings-app"

				Expect(cf.Cf("create-route", space, domain, "--hostname", hostname).Wait(defaultTimeout)).To(Exit(0))
				Expect(cf.Cf("map-route", app, domain, "--hostname", hostname).Wait(defaultTimeout)).To(Exit(0))
				Expect(cf.Cf("map-route", appTwo, domain, "--hostname", hostname).Wait(defaultTimeout)).To(Exit(0))
			})

			It("successfully load balances requests to the apps", func() {
				res, err := http.Get(appURL)
				Expect(err).ToNot(HaveOccurred())

				body, err := ioutil.ReadAll(res.Body)
				Expect(err).ToNot(HaveOccurred())

				type AppResponse struct {
					Greeting string `json:"greeting"`
				}

				var appOneResp AppResponse
				err = json.Unmarshal(body, &appOneResp)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() (AppResponse, error) {
					res, err := http.Get(appTwoURL)
					if err != nil {
						return AppResponse{}, err
					}

					body, err := ioutil.ReadAll(res.Body)
					if err != nil {
						return AppResponse{}, err
					}

					var appTwoResp AppResponse
					err = json.Unmarshal(body, &appTwoResp)
					if err != nil {
						return AppResponse{}, err
					}

					return appTwoResp, nil
				}, defaultTimeout, time.Second).ShouldNot(Equal(appOneResp))
			})
		})
	})
})

type Instance struct {
	Index string `json:"instance_index"`
	GUID  string `json:"instance_guid"`
}

func getAppResponse(resp io.ReadCloser) Instance {
	body, err := ioutil.ReadAll(resp)
	Expect(err).ToNot(HaveOccurred())

	var instance Instance
	err = json.Unmarshal(body, &instance)
	Expect(err).NotTo(HaveOccurred())
	return instance
}

func getStatusCode(appURL string) (int, error) {
	res, err := http.Get(appURL)
	if err != nil {
		return 0, err
	}
	return res.StatusCode, nil
}

func appGuid(a string) string {
	appGuidCmd := cf.Cf("app", a, "--guid")
	Expect(appGuidCmd.Wait(defaultTimeout)).To(Exit(0))
	appGuid := string(appGuidCmd.Out.Contents())
	return strings.TrimSuffix(appGuid, "\n")
}

func spaceGuid(s string) string {
	spaceGuidCmd := cf.Cf("space", s, "--guid")
	Expect(spaceGuidCmd.Wait(defaultTimeout)).To(Exit(0))
	spaceGuid := string(spaceGuidCmd.Out.Contents())
	return strings.TrimSuffix(spaceGuid, "\n")
}

func domainGuid(d string) string {
	domainGuidCmd := cf.Cf("curl", "/v2/domains?q=name:"+d)
	Expect(domainGuidCmd.Wait(defaultTimeout)).To(Exit(0))
	domainResp := string(domainGuidCmd.Out.Contents())
	return getEntityGuid(domainResp)
}

func routeGuid(space string, hostname string) string {
	routeGuidCmd := cf.Cf("curl", fmt.Sprintf("/v2/routes?q=host:%s", hostname))
	Expect(routeGuidCmd.Wait(defaultTimeout)).To(Exit(0))
	routeResp := string(routeGuidCmd.Out.Contents())
	return getEntityGuid(routeResp)
}

func getEntityGuid(s string) string {
	regex := regexp.MustCompile(`\s+"guid": "(.+)"`)
	return regex.FindStringSubmatch(s)[1]
}
