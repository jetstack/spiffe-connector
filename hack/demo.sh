#!/usr/bin/env bash

set -exuo pipefail

PROJECT=spiffe-connector
ARCH=$(go env GOARCH)
VERSION=$(git ls-files | xargs -n 1 cat | md5sum | head -c 7)
KUBECONFIG=./dist/kubeconfig

cat .goreleaser.demo.yaml | ARCH=$ARCH envsubst > .goreleaser.demo.$ARCH.yaml

VERSION=$VERSION goreleaser release -f .goreleaser.demo.$ARCH.yaml --snapshot --rm-dist

kind get clusters | grep $PROJECT || kind create cluster --name $PROJECT --image=kindest/node:v1.23.4

kind get kubeconfig --name spiffe-connector > ./dist/kubeconfig
export KUBECONFIG=./dist/kubeconfig

kind load docker-image --name $PROJECT "jetstack/spiffe-connector-server:$VERSION-$ARCH"
kind load docker-image --name $PROJECT "jetstack/spiffe-connector-sidecar:$VERSION-$ARCH"
kind load docker-image --name $PROJECT "jetstack/spiffe-connector-example:$VERSION-$ARCH"

# Install cert-manager
kubectl apply -f "./deploy/01-cert-manager.yaml"
until cmctl check api; do sleep 5; done

# install CSI driver and trust
kubectl apply -n cert-manager -f "./deploy/02-csi-driver-spiffe.yaml"
kubectl apply -n cert-manager -f "./deploy/03-trust.yaml"

while [ "$(kubectl get deployment -n cert-manager cert-manager-trust -o json | jq '.status.availableReplicas')" != "$(kubectl get deployment -n cert-manager cert-manager-trust -o json | jq '.spec.replicas')" ]
do
  echo "waiting for cm trust to start"
  sleep 1
done

# Approve Trust webhook serving certificate
sleep 2
for i in $(kubectl get cr -n cert-manager -o=jsonpath="{.items[*]['metadata.name']}"); do cmctl approve -n cert-manager $i || true ; done

until kubectl rollout status -n cert-manager deployment/cert-manager-trust ; do sleep 5; done

# Bootstrap a self-signed CA
kubectl apply -n cert-manager -f "./deploy/04-selfsigned-ca.yaml"

# Approve Trust Domain CertificateRequest
sleep 2
for i in $(kubectl get cr -n cert-manager -o=jsonpath="{.items[*]['metadata.name']}"); do cmctl approve -n cert-manager $i || true; done

# Prepare trust bundle
kubectl apply -n cert-manager -f "./deploy/05-trust-domain-bundle.yaml"

# Deploy the spiffe connector server
cat "./deploy/06-spiffe-connector-server.yaml" | \
  ARCH=$ARCH \
  VERSION=$VERSION \
  GOOGLE_CREDENTIALS=$(cat ~/.config/gcloud/application_default_credentials.json | awk '$0="    "$0') \
  AWS_CREDENTIALS=$(cat ~/.aws/credentials | awk '$0="    "$0') \
  envsubst | \
  kubectl apply -f -

while [ "$(kubectl get deployment -n spiffe-connector spiffe-connector -o json | jq '.status.availableReplicas')" != "$(kubectl get deployment -n spiffe-connector spiffe-connector -o json | jq '.spec.replicas')" ]
do
  echo "waiting for server to start"
  sleep 1
done

# Deploy example workload with spiffe-connector sidecar
cat "./deploy/07-example-app.yaml" | ARCH=$ARCH VERSION=$VERSION envsubst | kubectl apply -f -

# port-forward to the application's UI
while [ "$(kubectl get deployment -n example-app example-app -o json | jq '.status.availableReplicas')" != "$(kubectl get deployment -n example-app example-app -o json | jq '.spec.replicas')" ]
do
  echo "waiting for workload to start"
  sleep 1
done
kubectl port-forward -n example-app svc/example-app 3000
