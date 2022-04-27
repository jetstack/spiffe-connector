package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/jetstack/spiffe-connector/internal/pkg/cryptoutil"
)

const testConfig = `---
acls:
- match_principal: "spiffe://foo/bar/baz"
  credentials:
  - provider: "google"
    object_reference: "service-account@example.com"
  - provider: "aws"
    object_reference: "aws::arn:foo"
spiffe:
  svid_sources:
    files:
      trust_domain_ca: ./ca.pem
      svid_cert: ./svid_cert.pem
      svid_key: ./svid_key.pem
`

// Generate testing material for use locally.
func main() {
	if len(os.Args) < 2 {
		exitOnErr(errors.New("usage: " + os.Args[0] + " 'spiffe://your.domain/your/id'"))
	}
	certs, err := cryptoutil.GenerateTestCerts(os.Args[1])
	exitOnErr(err)
	leafCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certs.Certificate[0],
	})
	caCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certs.Certificate[1],
	})
	x509Key, _ := x509.MarshalPKCS8PrivateKey(certs.PrivateKey)
	key := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509Key,
	})
	exitOnErr(os.WriteFile("ca.pem", caCert, 0o600))
	exitOnErr(os.WriteFile("svid_cert.pem", leafCert, 0o600))
	exitOnErr(os.WriteFile("svid_key.pem", key, 0o666))
	exitOnErr(os.WriteFile("test.yaml", []byte(testConfig), 0o666))
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't generate certs: %s\n", err.Error())
		os.Exit(1)
	}
}
