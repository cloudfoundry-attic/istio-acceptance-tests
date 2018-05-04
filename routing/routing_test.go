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
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Routing", func() {
	var (
		domain            string
		app               string
		helloRoutingAsset = "../assets/golang"
		appURL            string
	)

	BeforeEach(func() {
		domain = IstioDomain()

		app = random_name.CATSRandomName("APP")
		pushCmd := cf.Cf("push", app,
			"-p", helloRoutingAsset,
			"-f", fmt.Sprintf("%s/manifest.yml", helloRoutingAsset),
			"-d", domain,
			"-i", "2").Wait(defaultTimeout)
		Expect(pushCmd).To(Exit(0))
		appURL = fmt.Sprintf("http://%s.%s", app, domain)

		Eventually(func() (int, error) {
			res, err := http.Get(appURL)
			if err != nil {
				return 0, err
			}
			return res.StatusCode, err
		}, defaultTimeout).Should(Equal(200))
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

			instanceOne := getAppResponse(res.Body)
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

	Context("when an app has many routes", func() {
		var (
			hostNameOne string
			hostNameTwo string
			space       string
		)

		BeforeEach(func() {
			tw := helpers.TestWorkspace{}
			space = tw.SpaceName()
			hostNameOne = "app1"
			hostNameTwo = "app2"

			createRouteOneCmd := cf.Cf("create-route", space, domain, "--hostname", hostNameOne)
			Expect(createRouteOneCmd.Wait(defaultTimeout)).To(Exit(0))
			createRouteTwoCmd := cf.Cf("create-route", space, domain, "--hostname", hostNameTwo)
			Expect(createRouteTwoCmd.Wait(defaultTimeout)).To(Exit(0))

			mapRouteOneCmd := cf.Cf("map-route", app, domain, "--hostname", hostNameOne)
			Expect(mapRouteOneCmd.Wait(defaultTimeout)).To(Exit(0))
			mapRouteTwoCmd := cf.Cf("map-route", app, domain, "--hostname", hostNameTwo)
			Expect(mapRouteTwoCmd.Wait(defaultTimeout)).To(Exit(0))
		})

		FContext("can map route using both CAPI API endpoints", func() {

			// TODO: try to put the common ops in both tests below in before each
			It("can map route using Apps API", func() {
				oauthToken := authToken()
				hostName := fmt.Sprintf("someApp-%d", time.Now().UnixNano)

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

				oauthToken = authToken()
				routeCreateReq.Header.Add("Authorization", oauthToken)
				routeCreateReq.Header.Set("Content-Type", "Application/json")

				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				client := &http.Client{Transport: tr}
				resp, err := client.Do(routeCreateReq)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				b, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				routeGuid := getEntityGuid(string(b))

				appGuidCmd := cf.Cf("app", app, "--guid")
				Expect(appGuidCmd.Wait(defaultTimeout)).To(Exit(0))
				appGuid := string(appGuidCmd.Out.Contents())
				appGuid = strings.TrimSuffix(appGuid, "\n")

				reqURI := fmt.Sprintf("https://api.%s/v2/apps/%s/routes/%s", SystemDomain(), appGuid, routeGuid)
				req, err := http.NewRequest("PUT", reqURI, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("Authorization", oauthToken)

				resp, err = client.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				defer resp.Body.Close()

				Eventually(func() (int, error) {
					appURLOne := fmt.Sprintf("http://%s.%s", hostName, domain)
					res, err := http.Get(appURLOne)
					if err != nil {
						return 0, err
					}
					return res.StatusCode, nil
				}, defaultTimeout, time.Second).Should(Equal(200))
			})

			It("can map route using Routes API", func() {
				oauthToken := authToken()
				hostName := fmt.Sprintf("someApp-%d", time.Now().UnixNano)

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

				oauthToken = authToken()
				routeCreateReq.Header.Add("Authorization", oauthToken)
				routeCreateReq.Header.Set("Content-Type", "Application/json")

				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				client := &http.Client{Transport: tr}
				resp, err := client.Do(routeCreateReq)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				b, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				routeGuid := getEntityGuid(string(b))

				appGuidCmd := cf.Cf("app", app, "--guid")
				Expect(appGuidCmd.Wait(defaultTimeout)).To(Exit(0))
				appGuid := string(appGuidCmd.Out.Contents())
				appGuid = strings.TrimSuffix(appGuid, "\n")

				reqURI := fmt.Sprintf("https://api.%s/v2/routes/%s/apps/%s", SystemDomain(), routeGuid, appGuid)
				req, err := http.NewRequest("PUT", reqURI, nil)
				Expect(err).NotTo(HaveOccurred())

				req.Header.Add("Authorization", oauthToken)

				resp, err = client.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))
				defer resp.Body.Close()

				Eventually(func() (int, error) {
					appURLOne := fmt.Sprintf("http://%s.%s", hostName, domain)
					res, err := http.Get(appURLOne)
					if err != nil {
						return 0, err
					}
					return res.StatusCode, nil
				}, defaultTimeout, time.Second).Should(Equal(200))
			})

		})

		It("requests succeed to all routes", func() {
			Eventually(func() (int, error) {
				appURLOne := fmt.Sprintf("http://%s.%s", hostNameOne, domain)
				res, err := http.Get(appURLOne)
				if err != nil {
					return 0, err
				}
				return res.StatusCode, nil
			}, defaultTimeout, time.Second).Should(Equal(200))

			Eventually(func() (int, error) {
				appURLTwo := fmt.Sprintf("http://%s.%s", hostNameTwo, domain)
				res, err := http.Get(appURLTwo)
				if err != nil {
					return 0, err
				}
				return res.StatusCode, nil
			}, defaultTimeout, time.Second).Should(Equal(200))
		})

		It("successfully unmaps routes and request continue to succeed for mapped routes", func() {
			unmapRouteOneCmd := cf.Cf("unmap-route", app, domain, "--hostname", hostNameOne)
			Expect(unmapRouteOneCmd.Wait(defaultTimeout)).To(Exit(0))

			Eventually(func() (int, error) {
				appURLOne := fmt.Sprintf("http://%s.%s", hostNameOne, domain)
				res, err := http.Get(appURLOne)
				if err != nil {
					return 0, err
				}
				return res.StatusCode, nil
			}, defaultTimeout).Should(Equal(404))

			Eventually(func() (int, error) {
				appURLTwo := fmt.Sprintf("http://%s.%s", hostNameTwo, domain)
				res, err := http.Get(appURLTwo)
				if err != nil {
					return 0, err
				}
				return res.StatusCode, nil
			}, defaultTimeout, time.Second).Should(Equal(200))
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
