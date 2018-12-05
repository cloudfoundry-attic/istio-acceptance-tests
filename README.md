# Istio Acceptance Tests
This repository contains code for demoing the [Istio bookinfo
app](https://istio.io/docs/guides/bookinfo.html) on Cloud Foundry and ensuring
that it works with new updates.

### Prerequisites
- Working installation of Go
- Valid `$GOPATH`

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
	"details_docker_tag": "istio/examples-bookinfo-details-v1:1.5.0"
	"wilcard_ca": "<envoy_wildcard_ca.ca>"
}
EOF
```

Note: `wildcard_ca` is an optional property. It should be be configured if TLS
is enabled between clients and the istio-router (enabled by using the
`enable-tls-termination` ops-file).

## Running Tests
```sh
CONFIG="$PWD/config.json" scripts/test
```
