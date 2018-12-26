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
	"cf_internal_istio_domain": "istio.apps.internal",
	"cf_istio_domain": "istio.<system-domain>",
  "product_page_docker_tag": "istio/examples-bookinfo-productpage-v1:1.8.0",
  "reviews_docker_tag": "xanderstrike/istio-bookinfo-reviews-v3:latest",
  "ratings_docker_tag": "istio/examples-bookinfo-ratings-v1:1.8.0",
  "details_docker_tag": "istio/examples-bookinfo-details-v1:1.8.0",
	"include_internal_route_tests": false,
	"wildcard_ca": "<envoy_wildcard_ca.ca>"
}
EOF
```

Note: `wildcard_ca` is an optional property. It should be be configured if TLS
is enabled between clients and the istio-router (enabled by using the
`enable-tls-termination` ops-file).

Note: `include_internal_route_tests` is an optional property. If set to true, the
internal route tests will run. This will require the Envoy sidecar to be in the
network datapath (enabled by using the `enable-sidecar-proxying` ops-file).

## Running Tests
```sh
CONFIG="$PWD/config.json" scripts/test
```
