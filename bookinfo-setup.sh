#!/bin/bash

# this script deploys a complete bookinfo demo for PMs and anyone else who'd like a demo

if [ -z "$1" ]; then
  echo "the first arg should be your system domain (for example istio.beanie.c2c.cf-app.com)"
  exit
fi

cf delete-org -f bookinfo

cf create-org bookinfo
cf t -o bookinfo
cf create-space s
cf t -o bookinfo -s s

# the only difference between the xanderstrike image and the istio one is this one comes with basic
# command line utilities like cURL for debugging purposes
cf push productpage -o xanderstrike/istio-productpage-fat:latest -d $1

cf push ratings -o istio/examples-bookinfo-ratings-v1:1.8.0 -d istio.apps.internal
cf push details -o istio/examples-bookinfo-details-v1:1.8.0 -d istio.apps.internal

# reviews has some problems
# - it doesn't respond to health checks for some reason
# - it binds both port 9080 and port 9443, but doesn't use the latter, and cloud foundry picks 9443
# - the xanderstrike images _only_ expose 9080, so they work with CF
cf push reviews-v1 -o xanderstrike/istio-bookinfo-reviews-v1:latest -u none --no-route
cf push reviews-v2 -o xanderstrike/istio-bookinfo-reviews-v2:latest -u none --no-route
cf push reviews-v3 -o xanderstrike/istio-bookinfo-reviews-v3:latest -u none --no-route

# cf map-route reviews-v1 istio.apps.internal --hostname reviews # this is commented because of the weighting bug
cf map-route reviews-v2 istio.apps.internal --hostname reviews
cf map-route reviews-v3 istio.apps.internal --hostname reviews

cf push ratings-db -o mongo:3.4 -d istio.apps.internal

cf add-network-policy productpage --destination-app details --protocol tcp --port 9080

cf add-network-policy productpage --destination-app reviews-v1 --protocol tcp --port 9080
cf add-network-policy productpage --destination-app reviews-v2 --protocol tcp --port 9080
cf add-network-policy productpage --destination-app reviews-v3 --protocol tcp --port 9080

# reviews-v1 does not call out to ratings service
cf add-network-policy reviews-v2 --destination-app ratings --protocol tcp --port 9080
cf add-network-policy reviews-v3 --destination-app ratings --protocol tcp --port 9080

cf add-network-policy ratings --destination-app ratings-db

cf set-env productpage SERVICES_DOMAIN istio.apps.internal
cf restage productpage

cf set-env ratings MONGO_DB_URL ratings-db.istio.apps.internal
cf restage ratings

# reviews is picky
cf set-env reviews-v1 SERVICES_DOMAIN istio.apps.internal
cf set-env reviews-v1 SERVICE_9080_NAME reviews-v1
cf set-env reviews-v1 SERVICE_TAGS version|v1
cf set-env reviews-v1 SERVICE_PROTOCOL http
cf set-env reviews-v1 SERVICE_9443_IGNORE 1
cf restage reviews-v1

cf set-env reviews-v2 SERVICES_DOMAIN istio.apps.internal
cf set-env reviews-v2 SERVICE_9080_NAME reviews-v2
cf set-env reviews-v2 SERVICE_TAGS version|v2
cf set-env reviews-v2 SERVICE_PROTOCOL http
cf set-env reviews-v2 SERVICE_9443_IGNORE 1
cf restage reviews-v2

cf set-env reviews-v3 SERVICES_DOMAIN istio.apps.internal
cf set-env reviews-v3 SERVICE_9080_NAME reviews-v3
cf set-env reviews-v3 SERVICE_TAGS version|v2
cf set-env reviews-v3 SERVICE_PROTOCOL http
cf set-env reviews-v3 SERVICE_9443_IGNORE 1
cf restage reviews-v3
