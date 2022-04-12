# spiffe-connector

Using SPIFFE Verifiable Identity Documents to seamlessly authenticate to existing services.


```yaml
- principal: "spiffe://foo/bar/baz"
  credentials:
  - provider: "google"
    object_reference: "service-account@example.com"
```
