package bookinfo_demo

import (
	"fmt"
	"os"
	"testing"
	"time"

	"code.cloudfoundry.org/istio-acceptance-tests/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/sclevine/agouti"
)

func TestBookinfoDemo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BookinfoDemo Suite")
}

var (
	agoutiDriver   *agouti.WebDriver
	c              config.Config
	defaultTimeout = 120 * time.Second
	org, space     = "ISTIO-ORG", "ISTIO-SPACE"
)

var _ = BeforeSuite(func() {
	var err error
	configPath := os.Getenv("CONFIG")
	Expect(configPath).NotTo(BeEmpty())
	fmt.Println(configPath)
	c, err = config.NewConfig(configPath)
	Expect(err).ToNot(HaveOccurred())

	apiCmd := cf.Cf("api", fmt.Sprintf("api.%s", c.ApiEndpoint), "--skip-ssl-validation").Wait(defaultTimeout)
	Expect(apiCmd).To(Exit(0))

	loginCmd := cf.Cf("auth", c.AdminUser, c.AdminPassword).Wait(defaultTimeout)
	Expect(loginCmd).To(Exit(0))

	enableDockerCmd := cf.Cf("enable-feature-flag", "diego_docker").Wait(defaultTimeout)
	Expect(enableDockerCmd).To(Exit(0))

	orgCmd := cf.Cf("create-org", org).Wait(defaultTimeout)
	Expect(orgCmd).To(Exit(0))
	spaceCmd := cf.Cf("create-space", "-o", org, space).Wait(defaultTimeout)
	Expect(spaceCmd).To(Exit(0))
	targetCmd := cf.Cf("target", "-o", org, "-s", space).Wait(defaultTimeout)
	Expect(targetCmd).To(Exit(0))

	productPagePush := cf.Cf("push", "productpage", "-o", c.ProductPageDockerWithTag, "-d", c.ApiEndpoint).Wait(defaultTimeout)
	Expect(productPagePush).To(Exit(0))
	ratingsPush := cf.Cf("push", "ratings", "-o", c.RatingsDockerWithTag, "-d", c.AppsDomain).Wait(defaultTimeout)
	Expect(ratingsPush).To(Exit(0))
	reviewsPush := cf.Cf("push", "reviews", "-o", c.ReviewsDockerWithTag, "-d", c.AppsDomain, "-u", "none").Wait(defaultTimeout)
	Expect(reviewsPush).To(Exit(0))
	detailsPush := cf.Cf("push", "details", "-o", c.DetailsDockerWithTag, "-d", c.AppsDomain).Wait(defaultTimeout)
	Expect(detailsPush).To(Exit(0))

	setProductEnvVar := cf.Cf("set-env", "productpage", "SERVICES_DOMAIN", c.AppsDomain).Wait(defaultTimeout)
	Expect(setProductEnvVar).To(Exit(0))
	productRestage := cf.Cf("restage", "productpage").Wait(defaultTimeout)
	Expect(productRestage).To(Exit(0))
	setReviewsEnvVar := cf.Cf("set-env", "reviews", "SERVICES_DOMAIN", c.AppsDomain).Wait(defaultTimeout)
	Expect(setReviewsEnvVar).To(Exit(0))
	reviewsRestage := cf.Cf("restage", "reviews").Wait(defaultTimeout)
	Expect(reviewsRestage).To(Exit(0))

	productDetailsPolicy := cf.Cf("add-network-policy", "productpage", "--destination-app", "details", "--protocol", "tcp", "--port", "9080").Wait(defaultTimeout)
	Expect(productDetailsPolicy).To(Exit(0))
	productReviewsPolicy := cf.Cf("add-network-policy", "productpage", "--destination-app", "reviews", "--protocol", "tcp", "--port", "9080").Wait(defaultTimeout)
	Expect(productReviewsPolicy).To(Exit(0))
	reviewsRatingsPolicy := cf.Cf("add-network-policy", "reviews", "--destination-app", "ratings", "--protocol", "tcp", "--port", "9080").Wait(defaultTimeout)
	Expect(reviewsRatingsPolicy).To(Exit(0))

	agoutiDriver = agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{
			"--headless",
			"--disable-gpu",
			"--allow-insecure-localhost",
			"--no-sandbox",
		}),
	)
	Expect(agoutiDriver.Start()).To(Succeed())
})

var _ = AfterSuite(func() {
	cleanUpProductPage := cf.Cf("delete", "productpage", "-f", "-r").Wait(defaultTimeout)
	Expect(cleanUpProductPage).To(Exit(0))
	cleanUpReviews := cf.Cf("delete", "reviews", "-f", "-r").Wait(defaultTimeout)
	Expect(cleanUpReviews).To(Exit(0))
	cleanUpRatings := cf.Cf("delete", "ratings", "-f", "-r").Wait(defaultTimeout)
	Expect(cleanUpRatings).To(Exit(0))
	cleanUpDetails := cf.Cf("delete", "details", "-f", "-r").Wait(defaultTimeout)
	Expect(cleanUpDetails).To(Exit(0))
	cleanUpCmd := cf.Cf("delete-org", org, "-f").Wait(defaultTimeout)
	Expect(cleanUpCmd).To(Exit(0))
	Expect(agoutiDriver.Stop()).To(Succeed())
})
