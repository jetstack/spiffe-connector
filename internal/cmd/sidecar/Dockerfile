FROM gcr.io/distroless/static
COPY spiffe-connector-sidecar /spiffe-connector-sidecar
USER 65532:65532 # nonroot
ENTRYPOINT ["/spiffe-connector-sidecar"]
