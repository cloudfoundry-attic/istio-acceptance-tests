package helpers

import (
	"code.cloudfoundry.org/istio-acceptance-tests/config"
)

type TestUser struct {
	config.Config
}

func (tu TestUser) Username() string {
	return tu.AdminUser
}

func (tu TestUser) Password() string {
	return tu.AdminPassword
}

type TestWorkspace struct {
	Org   string
	Space string
}

func (tw TestWorkspace) OrganizationName() string {
	return tw.Org
}

func (tw TestWorkspace) SpaceName() string {
	return tw.Space
}
