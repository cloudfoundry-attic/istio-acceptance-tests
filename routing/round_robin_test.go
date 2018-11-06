package routing_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Round Robin", func() {
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
		}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
	})

	Context("when the app has many instances", func() {
		BeforeEach(func() {
			Expect(cf.Cf("scale", app, "-i", "2").Wait(defaultTimeout)).To(Exit(0))

			Eventually(func() (int, error) {
				return getStatusCode(appURL)
			}, defaultTimeout, time.Second).Should(Equal(http.StatusOK))
		})

		It("successfully load balances between instances", func() {
			resp, err := http.Get(appURL)
			Expect(err).NotTo(HaveOccurred())

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

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
			holaRoutingDroplet = "../assets/hola-golang.tgz"
			appTwo             string
			appTwoURL          string
			hostname           string
		)

		BeforeEach(func() {
			appTwo = app + "-2"
			Expect(cf.Cf("push", appTwo,
				"-d", domain,
				"-s", "cflinuxfs3",
				"--droplet", holaRoutingDroplet,
				"-i", "1",
				"-m", "16M",
				"-k", "75M").Wait(defaultTimeout)).To(Exit(0))
			appTwoURL = fmt.Sprintf("http://%s.%s", appTwo, domain)

			Eventually(func() (int, error) {
				return getStatusCode(appTwoURL)
			}, defaultTimeout).Should(Equal(http.StatusOK))

			hostname = generator.PrefixedRandomName("greetings", "app")

			Expect(cf.Cf("create-route", spaceName(), domain, "--hostname", hostname).Wait(defaultTimeout)).To(Exit(0))
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
