package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	CFSystemDomain           string `json:"cf_system_domain"`
	CFInternalAppsDomain     string `json:"cf_internal_apps_domain"`
	IstioDomain              string `json:"cf_istio_domain"`
	AdminUser                string `json:"cf_admin_user"`
	AdminPassword            string `json:"cf_admin_password"`
	ProductPageDockerWithTag string `json:"product_page_docker_tag"`
	ReviewsDockerWithTag     string `json:"reviews_docker_tag"`
	RatingsDockerWithTag     string `json:"ratings_docker_tag"`
	DetailsDockerWithTag     string `json:"details_docker_tag"`
}

func NewConfig(path string) (Config, error) {
	var config Config

	configFile, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	return config, err
}

func (c Config) Validate() error {
	missingProperties := []string{}
	if c.IstioDomain == "" {
		missingProperties = append(missingProperties, "cf_istio_domain")
	}
	if c.CFSystemDomain == "" {
		missingProperties = append(missingProperties, "cf_system_domain")
	}
	if c.AdminUser == "" {
		missingProperties = append(missingProperties, "cf_admin_user")
	}
	if c.AdminPassword == "" {
		missingProperties = append(missingProperties, "cf_admin_password")
	}
	if c.CFInternalAppsDomain == "" {
		missingProperties = append(missingProperties, "cf_internal_apps_domain")
	}
	if c.IstioDomain == "" {
		missingProperties = append(missingProperties, "cf_istio_domain")
	}
	if c.ProductPageDockerWithTag == "" {
		c.ProductPageDockerWithTag = "istio/examples-bookinfo-productpage-v1:1.5.0"
	}
	if c.ReviewsDockerWithTag == "" {
		c.ReviewsDockerWithTag = "istio/examples-bookinfo-reviews-v3:1.5.0"
	}
	if c.RatingsDockerWithTag == "" {
		c.RatingsDockerWithTag = "istio/examples-bookinfo-ratings-v1:1.5.0"
	}
	if c.DetailsDockerWithTag == "" {
		c.DetailsDockerWithTag = "istio/examples-bookinfo-details-v1:1.5.0"
	}
	if len(missingProperties) > 0 {
		return errors.New(fmt.Sprintf("Missing required config properties: %s", strings.Join(missingProperties, ", ")))
	} else {
		return nil
	}
}
