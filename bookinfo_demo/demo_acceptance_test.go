package bookinfo_demo

import (
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
	})

	AfterEach(func() {
		Expect(page.Destroy()).To(Succeed())
	})

	var _ = Describe("Bookinfo Pages", func() {
		Context("Product Page", func() {
			BeforeEach(func() {
				Expect(page.Navigate("http://productpage.bosh-lite.com")).To(Succeed())
			})

			It("can be visited", func() {
				Eventually(page).Should(HaveTitle("Simple Bookstore App"))
				Eventually(page.Find("h3")).Should(HaveText("Hello! This is a simple bookstore application consisting of three services as shown below"))
			})

			It("has the correct internal addresses", func() {
				html, err := page.HTML()
				Expect(err).NotTo(HaveOccurred())
				Eventually(html).Should(ContainSubstring("http://details.apps.internal:9080"))
				Eventually(html).Should(ContainSubstring("http://reviews.apps.internal:9080"))
				Eventually(html).Should(ContainSubstring("http://ratings.apps.internal:9080"))
			})

			It("links to the normal and test users", func() {
				Expect(page.FindByLink("Normal user")).Should(BeFound())
				Expect(page.FindByLink("Test user")).Should(BeFound())
			})

			Context("Normal user", func() {
				var html string
				var err error

				BeforeEach(func() {
					Expect(page.FindByLink("Normal user").Click()).To(Succeed())

					html, err = page.HTML()
					Expect(err).NotTo(HaveOccurred())
				})

				It("navigates to the product page for the Comedy of Errors", func() {
					Eventually(html).Should(ContainSubstring("The Comedy of Errors"))
				})

				It("Displays details successfully", func() {
					Eventually(html).Should(ContainSubstring("1234567890"))
				})

				It("Displays reviews successfully", func() {
					Eventually(html).Should(ContainSubstring("An extremely entertaining play by Shakespeare."))
				})

				It("Displays red stars successfully", func() {
					Eventually(html).Should(ContainSubstring(`font color="red"`))
					Eventually(html).Should(ContainSubstring("glyphicon glyphicon-star"))
				})
			})
		})
	})
})
