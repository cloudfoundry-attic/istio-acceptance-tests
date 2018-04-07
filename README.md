# Istio Acceptance Tests
This repository contains code for demoing the [Istio bookinfo
app](https://istio.io/docs/guides/bookinfo.html) on Cloud Foundry and ensuring
that it works with new updates

### Prerequisites
- Go should be installed and in the PATH
- GOPATH should be set as described in http://golang.org/doc/code.html

### Setup

# deploy a bosh-lite ?Just bosh? with service discovery

# ?Vendor Packages?

# create a config file
'{  
	"cf_api": "bosh-lite.com",
	"cf_admin_user": "admin",
	"cf_admin_password": <Admin password from the bosh deployment>,
	"cf_apps_domain": "apps.internal",
	"product_page_docker_tag": "<dockerhub username>/<productpage image>:<version tag>",
	"reviews_docker_tag": "<dockerhub username>/<reviews image>:<version tag>",
	"ratings_docker_tag": "<dockerhub username>/<ratings image>:<version tag>",
	"details_docker_tag": "<dockerhub username>/<details image>:<version tag>",
}'

-	Current working username: zlav
- Current working images
	- examples-bookinfo-productpage-v1:1.0.0
	- examples-bookinfo-reviews-v3:3.0.0
	- examples-bookinfo-ratings-v1:1.0.0
	- examples-bookinfo-details-v1:1.0.0

## Demo the productpage

Run the script we havent built

## Running Tests



# run all tests
'CONFIG="$PWD/config.json"'
