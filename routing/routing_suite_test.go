package routing_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"code.cloudfoundry.org/istio-acceptance-tests/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	DEFAULT_MEMORY_LIMIT = "256M"
)

var (
	Config         config.Config
	TestSetup      *workflowhelpers.ReproducibleTestSuiteSetup
	defaultTimeout = 240 * time.Second
)

func TestRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	var err error
	configPath := os.Getenv("CONFIG")
	Expect(configPath).NotTo(BeEmpty())
	fmt.Println(configPath)
	Config, err = config.NewConfig(configPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(Config.Validate()).To(Succeed())

	var _ = BeforeSuite(func() {
		TestSetup = workflowhelpers.NewTestSuiteSetup(Config)
		TestSetup.Setup()
	})

	var _ = AfterSuite(func() {
		if TestSetup != nil {
			TestSetup.Teardown()
		}
	})

	RunSpecs(t, "Routing Suite")
}

func internalDomain() string {
	return Config.CFInternalAppsDomain
}

func istioDomain() string {
	return Config.IstioDomain
}

func systemDomain() string {
	return Config.CFSystemDomain
}
