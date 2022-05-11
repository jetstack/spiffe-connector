#!/usr/bin/env bash

set -exuo pipefail

PROJECT=spiffe-connector
ARCH=$(go env GOARCH)
VERSION=$(git ls-files | xargs -n 1 cat | md5sum | head -c 7)
KUBECONFIG=./dist/kubeconfig

# build the application and images with goreleaser
cat .goreleaser.demo.yaml | ARCH=$ARCH envsubst > .goreleaser.demo.$ARCH.yaml
VERSION=$VERSION goreleaser release -f .goreleaser.demo.$ARCH.yaml --snapshot --rm-dist

# create a new kind cluster and connect to it
kind get clusters | grep $PROJECT || kind create cluster --name $PROJECT --image=kindest/node:v1.23.4
kind get kubeconfig --name spiffe-connector > ./dist/kubeconfig
export KUBECONFIG=./dist/kubeconfig

# load all the images used in dependencies
images=(
  "quay.io/jetstack/cert-manager-controller:v1.6.1" \
  "quay.io/jetstack/cert-manager-cainjector:v1.6.1" \
  "quay.io/jetstack/cert-manager-webhook:v1.6.1" \
  "k8s.gcr.io/sig-storage/csi-node-driver-registrar:v2.5.0" \
  "k8s.gcr.io/sig-storage/livenessprobe:v2.6.0" \
  "quay.io/jetstack/cert-manager-csi-driver-spiffe:v0.2.0" \
  "quay.io/jetstack/cert-manager-csi-driver-spiffe-approver:v0.2.0" \
  "quay.io/jetstack/cert-manager-trust:v0.1.0" \
)
for image in "${images[@]}"
do
  echo preloading $image
  if [ -z "$(docker images -q $image)" ]; then
    docker pull $image
  fi
  kind load docker-image --name $PROJECT quay.io/jetstack/cert-manager-controller:v1.6.1
done

# load the demo images
kind load docker-image --name $PROJECT "jetstack/spiffe-connector-server:$VERSION-$ARCH"
kind load docker-image --name $PROJECT "jetstack/spiffe-connector-sidecar:$VERSION-$ARCH"
kind load docker-image --name $PROJECT "jetstack/spiffe-connector-example:$VERSION-$ARCH"

# Install cert-manager
kubectl apply -f "./deploy/01-cert-manager.yaml"
until cmctl check api; do sleep 5; done

# install CSI driver and trust
kubectl apply -n cert-manager -f "./deploy/02-csi-driver-spiffe.yaml"
kubectl apply -n cert-manager -f "./deploy/03-trust.yaml"
sleep 2
for i in $(kubectl get cr -n cert-manager -o=jsonpath="{.items[*]['metadata.name']}"); do cmctl approve -n cert-manager $i || true ; done

while [ "$(kubectl get deployment -n cert-manager cert-manager-trust -o json | jq '.status.availableReplicas')" != "$(kubectl get deployment -n cert-manager cert-manager-trust -o json | jq '.spec.replicas')" ]
do
  echo "waiting for cm trust to start"
  sleep 1
done

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
