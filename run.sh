#!/usr/bin/env bash

set -euxo pipefail

export CLOUDSDK_CORE_PROJECT=gcping-1369

curl -H "Authorization: Bearer $(gcloud auth print-access-token)" https://run.googleapis.com/v1/projects/${CLOUDSDK_CORE_PROJECT}/locations | jq .locations[].locationId | sed -e 's/"//g' > run_regions.txt

# Build the image.
image=$(KO_DOCKER_REPO=gcr.io/${CLOUDSDK_CORE_PROJECT} ko publish -B ./cmd/ping/)

while read region; do
  gcloud beta run deploy ping \
    --platform=managed \
    --region=${region} \
    --allow-unauthenticated \
    --update-env-vars=REGION=${region} \
    --image=${image} || echo $region is not a Cloud Run region
done < run_regions.txt
