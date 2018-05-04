package routing_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
		helloRoutingAsset = "../assets/hello-golang"
		appURL            string
	)

	BeforeEach(func() {
		domain = IstioDomain()

		app = random_name.CATSRandomName("APP")
		pushCmd := cf.Cf("push", app,
			"-p", helloRoutingAsset,
			"-f", fmt.Sprintf("%s/manifest.yml", helloRoutingAsset),
			"-d", domain,
			"-i", "1").Wait(defaultTimeout)
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

	Context("when an app has many routes", func() {
		var (
			hostnameOne string
			hostnameTwo string
		)

		BeforeEach(func() {
			tw := helpers.TestWorkspace{}
			space := tw.SpaceName()
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
				res, err := http.Get(appURLOne)
				if err != nil {
					return 0, err
				}
				return res.StatusCode, nil
			}, defaultTimeout, time.Second).Should(Equal(200))

			Eventually(func() (int, error) {
				appURLTwo := fmt.Sprintf("http://%s.%s", hostnameTwo, domain)
				res, err := http.Get(appURLTwo)
				if err != nil {
					return 0, err
				}
				return res.StatusCode, nil
			}, defaultTimeout, time.Second).Should(Equal(200))
		})

		It("successfully unmaps routes and request continue to succeed for mapped routes", func() {
			unmapRouteOneCmd := cf.Cf("unmap-route", app, domain, "--hostname", hostnameOne)
			Expect(unmapRouteOneCmd.Wait(defaultTimeout)).To(Exit(0))

			Eventually(func() (int, error) {
				appURLOne := fmt.Sprintf("http://%s.%s", hostnameOne, domain)
				res, err := http.Get(appURLOne)
				if err != nil {
					return 0, err
				}
				return res.StatusCode, nil
			}, defaultTimeout).Should(Equal(404))

			Eventually(func() (int, error) {
				appURLTwo := fmt.Sprintf("http://%s.%s", hostnameTwo, domain)
				res, err := http.Get(appURLTwo)
				if err != nil {
					return 0, err
				}
				return res.StatusCode, nil
			}, defaultTimeout, time.Second).Should(Equal(200))
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

				appTwo = random_name.CATSRandomName("APP")
				pushCmd := cf.Cf("push", appTwo,
					"-p", holaRoutingAsset,
					"-f", fmt.Sprintf("%s/manifest.yml", holaRoutingAsset),
					"-d", domain,
					"-i", "1").Wait(defaultTimeout)
				Expect(pushCmd).To(Exit(0))
				appTwoURL = fmt.Sprintf("http://%s.%s", appTwo, domain)

				Eventually(func() (int, error) {
					res, err := http.Get(appTwoURL)
					if err != nil {
						return 0, err
					}
					return res.StatusCode, err
				}, defaultTimeout).Should(Equal(200))

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
