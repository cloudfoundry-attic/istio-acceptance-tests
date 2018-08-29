package routing_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/istio-acceptance-tests/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
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

	RunSpecs(t, "Routing Suite")
}

var _ = BeforeSuite(func() {
	var err error
	configPath := os.Getenv("CONFIG")
	Expect(configPath).NotTo(BeEmpty())
	fmt.Println(configPath)
	Config, err = config.NewConfig(configPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(Config.Validate()).To(Succeed())
	if Config.CFInternalAppsDomain == "" {
		createCmd := cf.Cf("curl", "/v2/shared_domains", "-d", fmt.Sprintf("{\"name\": \"%s\", \"internal\": true}", config.DefaultInternalAppsDomain))
		Expect(createCmd.Wait(defaultTimeout)).To(Exit(0))
	}

	TestSetup = workflowhelpers.NewTestSuiteSetup(Config)
	TestSetup.Setup()
})

var _ = AfterSuite(func() {
	if TestSetup != nil {
		TestSetup.Teardown()
	}
})

func adminUserContext() workflowhelpers.UserContext {
	return TestSetup.AdminUserContext()
}

func spaceName() string {
	return TestSetup.TestSpace.SpaceName()
}

func organizationName() string {
	return TestSetup.TestSpace.OrganizationName()
}

func internalDomain() string {
	if Config.CFInternalAppsDomain == "" {
		return config.DefaultInternalAppsDomain
	}
	return Config.CFInternalAppsDomain
}

func istioDomain() string {
	return Config.IstioDomain
}

func systemDomain() string {
	return Config.CFSystemDomain
}

func getAppResponse(resp io.ReadCloser) Instance {
	body, err := ioutil.ReadAll(resp)
	Expect(err).ToNot(HaveOccurred())

	var instance Instance
	err = json.Unmarshal(body, &instance)
	Expect(err).NotTo(HaveOccurred())
	return instance
}

func getStatusCode(appURL string) (int, error) {
	res, err := http.Get(appURL)
	if err != nil {
		return 0, err
	}
	res.Body.Close()
	return res.StatusCode, nil
}

func applicationGuid(a string) string {
	appGuidCmd := cf.Cf("app", a, "--guid")
	Expect(appGuidCmd.Wait(defaultTimeout)).To(Exit(0))
	appGuid := string(appGuidCmd.Out.Contents())
	return strings.TrimSuffix(appGuid, "\n")
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

func routeGuid(space string, hostname string) string {
	routeGuidCmd := cf.Cf("curl", fmt.Sprintf("/v2/routes?q=host:%s", hostname))
	Expect(routeGuidCmd.Wait(defaultTimeout)).To(Exit(0))
	routeResp := string(routeGuidCmd.Out.Contents())
	return getEntityGuid(routeResp)
}

func getEntityGuid(s string) string {
	regex := regexp.MustCompile(`\s+"guid": "(.+)"`)
	return regex.FindStringSubmatch(s)[1]
}
