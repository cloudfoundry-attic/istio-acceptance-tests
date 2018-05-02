package helpers_test

import (
	"github.com/pivotal-cf-experimental/cf-test-helpers/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Context", func() {
	It("builds", func() {})

	Describe("NewContext", func() {
		var (
			config  helpers.Config
			context *helpers.ConfiguredContext
		)

		JustBeforeEach(func() {
			context = helpers.NewContext(config)
		})

		Context("when setting up context with an existing user", func() {
			BeforeEach(func() {
				config = configWithExistingUser()
			})

			It("uses the password of the existing user", func() {
				Expect(context.GetConfiguredPassword()).To(Equal("existing-pass"))
			})
		})

		Context("when setting up context with a new user", func() {
			Context("when a password is explicitly configured", func() {
				BeforeEach(func() {
					config = configWithNewUserAndConfiguredPassword("i-really-want-this-pass")
				})

				It("uses the configured password", func() {
					Expect(context.GetConfiguredPassword()).To(Equal("i-really-want-this-pass"))
				})
			})

			Context("when a password is not explicitly configured", func() {
				var (
					subsequentContext *helpers.ConfiguredContext
				)

				BeforeEach(func() {
					config = helpers.Config{}
				})

				JustBeforeEach(func() {
					subsequentContext = helpers.NewContext(config)
				})

				It("randomly generates a password", func() {
					Expect(context.GetConfiguredPassword()).NotTo(Equal(subsequentContext.GetConfiguredPassword()))
				})
			})
		})
	})
})

func configWithExistingUser() helpers.Config {
	return helpers.Config{
		UseExistingUser:      true,
		ExistingUser:         "existing-user",
		ExistingUserPassword: "existing-pass",
	}
}

func configWithNewUserAndConfiguredPassword(password string) helpers.Config {
	return helpers.Config{
		ConfigurableTestPassword: "i-really-want-this-pass",
	}
}
