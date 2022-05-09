FROM scratch

COPY spiffe-connector-example /spiffe-connector-example
# cacert.pem is an extra file included by our goreleaser configuration
COPY cacert.pem /etc/ssl/certs/ca-certificates.crt

USER 65534:65534
ENTRYPOINT ["/spiffe-connector-example"]
