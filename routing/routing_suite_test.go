package routing_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"code.cloudfoundry.org/istio-acceptance-tests/config"
	"code.cloudfoundry.org/istio-acceptance-tests/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func TestRouting(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Routing Suite")
}

const (
	DEFAULT_MEMORY_LIMIT = "256M"
)

var (
	c              config.Config
	defaultTimeout = 120 * time.Second
)

func IstioDomain() string {
	return c.IstioDomain
}

var _ = BeforeSuite(func() {
	var err error
	configPath := os.Getenv("CONFIG")
	Expect(configPath).NotTo(BeEmpty())
	fmt.Println(configPath)
	c, err = config.NewConfig(configPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(c.Validate()).To(Succeed())

	tw := helpers.TestWorkspace{}

	uc := workflowhelpers.NewUserContext(fmt.Sprintf("api.%s", c.CFSystemDomain), helpers.TestUser{c}, tw, true, defaultTimeout)
	uc.Login()

	orgCmd := cf.Cf("create-org", tw.OrganizationName()).Wait(defaultTimeout)
	Expect(orgCmd).To(Exit(0))
	spaceCmd := cf.Cf("create-space", "-o", tw.OrganizationName(), tw.SpaceName()).Wait(defaultTimeout)
	Expect(spaceCmd).To(Exit(0))

	uc.TargetSpace()
})