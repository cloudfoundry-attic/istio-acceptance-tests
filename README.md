# Istio Acceptance Tests
This repository contains code for demoing the [Istio bookinfo
app](https://istio.io/docs/guides/bookinfo.html) on Cloud Foundry and ensuring
that it works with new updates

### Prerequisites
- Working installation of Go
- Valid `$GOPATH`

# Deploy a bosh with
- bosh-dns
  - [use-bosh-dns.yml](https://github.com/cloudfoundry/cf-deployment/blob/master/operations/experimental/use-bosh-dns.yml)
- Service discovery enabled (ops-files below)
  - [use-bosh-dns-for-containers.yml](https://github.com/cloudfoundry/cf-deployment/blob/master/operations/experimental/use-bosh-dns-for-containers.yml)
  - [enable-service-discovery.yml](https://github.com/cloudfoundry/cf-deployment/blob/master/operations/experimental/enable-service-discovery.yml)


# Create a config file
```sh
cat << EOF > "${PWD}/config.json"
{
	"cf_system_domain": "bosh-lite.com",
	"cf_admin_user": "admin",
	"cf_admin_password": <admin password>,
	"cf_internal_apps_domain": "apps.internal",
	"cf_istio_domain": "istio.<system-domain>",
	"product_page_docker_tag": "cfrouting/examples-bookinfo-productpage-v1:latest",
	"reviews_docker_tag": "cfrouting/examples-bookinfo-reviews-v3:latest",
	"ratings_docker_tag": "istio/examples-bookinfo-ratings-v1:1.5.0",
	"details_docker_tag": "istio/examples-bookinfo-details-v1:1.5.0",
}
EOF
```

## Running Tests
```sh
CONFIG="$PWD/config.json" scripts/test
```
