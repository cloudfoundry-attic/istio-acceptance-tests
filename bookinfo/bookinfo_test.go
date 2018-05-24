package bookinfo

import (
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/istio-acceptance-tests/config"
	"github.com/sclevine/agouti"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/sclevine/agouti/matchers"
)

var _ = Describe("Bookinfo", func() {
	var (
		page *agouti.Page
		c    config.Config
	)

	BeforeEach(func() {
		var err error
		page, err = agoutiDriver.NewPage()
		Expect(err).NotTo(HaveOccurred())
		SetDefaultEventuallyPollingInterval(3 * time.Second)
		SetDefaultEventuallyTimeout(20 * time.Second)

		configPath := os.Getenv("CONFIG")
		Expect(configPath).NotTo(BeEmpty())

		c, err = config.NewConfig(configPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.Validate()).To(Succeed())
	})

	AfterEach(func() {
		Expect(page.Destroy()).To(Succeed())
	})

	var _ = Describe("Bookinfo Pages", func() {
		Context("Product Page", func() {
			BeforeEach(func() {
				productPage := fmt.Sprintf("http://productpage.%s", c.IstioDomain)
				Expect(page.Navigate(productPage)).To(Succeed())
				Eventually(load(page), defaultTimeout, time.Second).Should(ContainSubstring("Simple Bookstore App"))
			})

			It("has the correct content", func() {
				Expect(page.FindByLink("Normal user")).To(BeFound())
				Expect(page.FindByLink("Test user")).To(BeFound())

				Expect(page).To(HaveTitle("Simple Bookstore App"))
				Expect(page.Find("h3")).To(HaveText("Hello! This is a simple bookstore application consisting of three services as shown below"))
			})

			It("has the correct internal addresses", func() {
				html, err := page.HTML()
				Expect(err).NotTo(HaveOccurred())

				internalDomain := c.CFInternalAppsDomain

				Expect(html).To(ContainSubstring(fmt.Sprintf("http://details.%s:9080", internalDomain)))
				Expect(html).To(ContainSubstring(fmt.Sprintf("http://reviews.%s:9080", internalDomain)))
				Expect(html).To(ContainSubstring(fmt.Sprintf("http://ratings.%s:9080", internalDomain)))
			})

			Context("Normal user", func() {
				BeforeEach(func() {
					Expect(page.FindByLink("Normal user").Click()).To(Succeed())
				})

				It("navigates to the product page for the Comedy of Errors", func() {
					Eventually(load(page), defaultTimeout, time.Second).Should(ContainSubstring("The Comedy of Errors"))
				})

				It("displays details successfully", func() {
					Eventually(load(page), defaultTimeout, time.Second).Should(ContainSubstring("1234567890"))
				})

				It("displays reviews successfully", func() {
					Eventually(load(page), defaultTimeout, time.Second).Should(ContainSubstring("An extremely entertaining play by Shakespeare."))
				})

				It("displays red ratings successfully", func() {
					Eventually(load(page), defaultTimeout, time.Second).Should(ContainSubstring(`font color="red"`))
					Eventually(load(page), defaultTimeout, time.Second).Should(ContainSubstring("glyphicon glyphicon-star"))
				})
			})
		})
	})
})

func load(page *agouti.Page) func() string {
	return func() string {
		err := page.Refresh()
		Expect(err).NotTo(HaveOccurred())
		html, err := page.HTML()
		Expect(err).NotTo(HaveOccurred())
		return html
	}
}
