package routing_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	routing_helpers "code.cloudfoundry.org/cf-routing-test-helpers/helpers"
	"code.cloudfoundry.org/istio-acceptance-tests/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
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
		domain = IstioDomain()

		app = generator.PrefixedRandomName("IATS", "APP")
		pushCmd := cf.Cf("push", app,
			"-p", helloRoutingAsset,
			"-f", fmt.Sprintf("%s/manifest.yml", helloRoutingAsset),
			"-d", domain,
			"-i", "1").Wait(defaultTimeout)
		Expect(pushCmd).To(Exit(0))
		appURL = fmt.Sprintf("http://%s.%s", app, domain)

		Eventually(func() (int, error) {
			return getStatusCode(appURL)
		}, defaultTimeout).Should(Equal(http.StatusOK))
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
				return getStatusCode(appURL)
			}, defaultTimeout).Should(Equal(http.StatusServiceUnavailable))
		})
	})

	Context("when an app has many routes", func() {
		var (
			hostnameOne string
			hostnameTwo string
			space       string
			org         string
		)

		BeforeEach(func() {
			tw := helpers.TestWorkspace{}
			space = tw.SpaceName()
			org = tw.OrganizationName()
			hostnameOne = "app1"
			hostnameTwo = "app2"

			createRouteOneCmd := cf.Cf("create-route", space, domain, "--hostname", hostnameOne)
			Expect(createRouteOneCmd.Wait(defaultTimeout)).To(Exit(0))
			createRouteTwoCmd := cf.Cf("create-route", space, domain, "--hostname", hostnameTwo)
			Expect(createRouteTwoCmd.Wait(defaultTimeout)).To(Exit(0))

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
			}, defaultTimeout).Should(Equal(404))

			Eventually(func() (int, error) {
				appURLTwo := fmt.Sprintf("http://%s.%s", hostnameTwo, domain)
				return getStatusCode(appURLTwo)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
		})
	})

	Context("route mappings", func() {
		var (
			hostname   string
			space      string
			org        string
			oauthToken string
			client     *http.Client
			aGuid      string
		)

		BeforeEach(func() {
			tw := helpers.TestWorkspace{}
			space = tw.SpaceName()
			org = tw.OrganizationName()
			hostname = fmt.Sprintf("someApp-%d", time.Now().UnixNano)
			oauthToken = authToken()
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client = &http.Client{Transport: tr}
			aGuid = appGuid(app)
		})

		It("can map a route with a private domain", func() {
			privateDomain := fmt.Sprintf("%s.%s", generator.PrefixedRandomName("iats", "private"), domain)
			privateDomainGuidCmd := cf.Cf("create-domain", org, privateDomain)
			Expect(privateDomainGuidCmd.Wait(defaultTimeout)).To(Exit(0))

			privateHostname := fmt.Sprintf("someApp-%d", time.Now().UnixNano)
			mapRouteCmd := cf.Cf("map-route", app, privateDomain, "--hostname", privateHostname)
			Expect(mapRouteCmd.Wait(defaultTimeout)).To(Exit(0))

			Eventually(func() (int, error) {
				appURL := fmt.Sprintf("http://%s.%s", privateHostname, privateDomain)
				return getStatusCode(appURL)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
		})

		Context("mapping a route using both CAPI endpoints", func() {
			It("can map route using Apps API", func() {
				routeGuid := routeGuid(space, domain, hostname, oauthToken, client)
				reqURI := fmt.Sprintf("https://api.%s/v2/apps/%s/routes/%s", SystemDomain(), aGuid, routeGuid)
				req, err := http.NewRequest("PUT", reqURI, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("Authorization", oauthToken)

				resp, err := client.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				defer resp.Body.Close()

				Eventually(func() (int, error) {
					appURL := fmt.Sprintf("http://%s.%s", hostname, domain)
					return getStatusCode(appURL)
				}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
			})

			It("can map route using Routes API", func() {
				routeGuid := routeGuid(space, domain, hostname, oauthToken, client)
				reqURI := fmt.Sprintf("https://api.%s/v2/routes/%s/apps/%s", SystemDomain(), routeGuid, aGuid)
				req, err := http.NewRequest("PUT", reqURI, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("Authorization", oauthToken)

				resp, err := client.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				defer resp.Body.Close()

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
				scaleCmd := cf.Cf("scale", app, "-i", "2").Wait(defaultTimeout)
				Expect(scaleCmd).To(Exit(0))
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
				domain = IstioDomain()

				appTwo = generator.PrefixedRandomName("IATS", "APP")
				pushCmd := cf.Cf("push", appTwo,
					"-p", holaRoutingAsset,
					"-f", fmt.Sprintf("%s/manifest.yml", holaRoutingAsset),
					"-d", domain,
					"-i", "1").Wait(defaultTimeout)
				Expect(pushCmd).To(Exit(0))
				appTwoURL = fmt.Sprintf("http://%s.%s", appTwo, domain)

				Eventually(func() (int, error) {
					return getStatusCode(appTwoURL)
				}, defaultTimeout).Should(Equal(http.StatusOK))

				tw := helpers.TestWorkspace{}
				space := tw.SpaceName()
				hostname = "greetings-app"

				createRouteOneCmd := cf.Cf("create-route", space, domain, "--hostname", hostname)
				Expect(createRouteOneCmd.Wait(defaultTimeout)).To(Exit(0))

				mapRouteOneCmd := cf.Cf("map-route", app, domain, "--hostname", hostname)
				Expect(mapRouteOneCmd.Wait(defaultTimeout)).To(Exit(0))

				mapRouteTwoCmd := cf.Cf("map-route", appTwo, domain, "--hostname", hostname)
				Expect(mapRouteTwoCmd.Wait(defaultTimeout)).To(Exit(0))
			})

			AfterEach(func() {
				routing_helpers.AppReport(appTwo, defaultTimeout)
				routing_helpers.DeleteApp(appTwo, defaultTimeout)
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

func routeGuid(space string, domain string, hostName string, authToken string, client *http.Client) string {
	spaceGuid := spaceGuid(space)
	domainGuid := domainGuid(domain)

	postBody := map[string]string{
		"domain_guid": domainGuid,
		"space_guid":  spaceGuid,
		"host":        hostName,
	}

	jsonBody, err := json.Marshal(postBody)
	Expect(err).NotTo(HaveOccurred())

	// create a route not using cf helpers, since it does not return the reponse body
	// and -v returns too much stuff
	routeCreateReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.%s/v2/routes", SystemDomain()),
		bytes.NewBuffer(jsonBody),
	)

	routeCreateReq.Header.Add("Authorization", authToken)
	routeCreateReq.Header.Set("Content-Type", "Application/json")

	resp, err := client.Do(routeCreateReq)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	return getEntityGuid(string(b))
}

func getEntityGuid(s string) string {
	regex := regexp.MustCompile(`\s+"guid": "(.+)"`)
	return regex.FindStringSubmatch(s)[1]
}

func authToken() string {
	authTokenCmd := cf.Cf("oauth-token")
	Expect(authTokenCmd.Wait(defaultTimeout)).To(Exit(0))
	oauthToken := string(authTokenCmd.Out.Contents())
	return strings.TrimSuffix(oauthToken, "\n")
}
