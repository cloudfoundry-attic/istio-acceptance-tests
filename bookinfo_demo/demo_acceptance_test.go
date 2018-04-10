package bookinfo_demo

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
	. "github.com/sclevine/agouti/matchers"
)

var _ = Describe("DemoAcceptance", func() {
	var page *agouti.Page

	BeforeEach(func() {
		var err error
		page, err = agoutiDriver.NewPage()
		Expect(err).NotTo(HaveOccurred())
		SetDefaultEventuallyPollingInterval(3 * time.Second)
		SetDefaultEventuallyTimeout(20 * time.Second)
	})

	AfterEach(func() {
		Expect(page.Destroy()).To(Succeed())
	})

	var _ = Describe("Bookinfo Pages", func() {
		Context("Product Page", func() {
			BeforeEach(func() {
				Expect(page.Navigate(fmt.Sprintf("http://productpage.%s", os.Getenv("API_DOMAIN")))).To(Succeed())
			})

			It("can be visited", func() {
				Expect(page).To(HaveTitle("Simple Bookstore App"))
				Expect(page.Find("h3")).To(HaveText("Hello! This is a simple bookstore application consisting of three services as shown below"))
			})

			It("has the correct internal addresses", func() {
				html, err := page.HTML()
				Expect(err).NotTo(HaveOccurred())

				internalDomain := os.Getenv("INTERNAL_DOMAIN")

				Expect(html).To(ContainSubstring(fmt.Sprintf("http://details.%s:9080", internalDomain)))
				Expect(html).To(ContainSubstring(fmt.Sprintf("http://reviews.%s:9080", internalDomain)))
				Expect(html).To(ContainSubstring(fmt.Sprintf("http://ratings.%s:9080", internalDomain)))
			})

			It("links to the normal and test users", func() {
				Expect(page.FindByLink("Normal user")).Should(BeFound())
				Expect(page.FindByLink("Test user")).Should(BeFound())
			})

			Context("Normal user", func() {
				BeforeEach(func() {
					Expect(page.FindByLink("Normal user").Click()).To(Succeed())
				})

				It("navigates to the product page for the Comedy of Errors", func() {
					Eventually(load(page)).Should(ContainSubstring("The Comedy of Errors"))
				})

				It("displays details successfully", func() {
					Eventually(load(page)).Should(ContainSubstring("1234567890"))
				})

				It("displays reviews successfully", func() {
					Eventually(load(page)).Should(ContainSubstring("An extremely entertaining play by Shakespeare."))
				})

				It("displays red ratings successfully", func() {
					Eventually(load(page)).Should(ContainSubstring(`font color="red"`))
					Eventually(load(page)).Should(ContainSubstring("glyphicon glyphicon-star"))
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
