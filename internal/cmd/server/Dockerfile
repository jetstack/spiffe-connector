FROM scratch

COPY spiffe-connector-server /spiffe-connector-server
# cacert.pem is an extra file included by our goreleaser configuration
COPY cacert.pem /etc/ssl/certs/ca-certificates.crt

USER 65534:65534
ENTRYPOINT ["/spiffe-connector-server"]
