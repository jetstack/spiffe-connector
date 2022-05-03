FROM gcr.io/distroless/static
COPY spiffe-connector-server /spiffe-connector-server
USER 65532:65532 # nonroot
ENTRYPOINT ["/spiffe-connector-server"]
