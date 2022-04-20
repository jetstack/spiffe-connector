package cryptoutil

import (
	"crypto/x509"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSelfSignedServingCert(t *testing.T) {
	cert, err := SelfSignedServingCert()
	assert.NoError(t, err, "SelfSignedServingCert should not error")
	roots := x509.NewCertPool()
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	assert.NoError(t, err, "Couldn't parse self signed cert")
	roots.AddCert(x509Cert)
	_, err = x509Cert.Verify(x509.VerifyOptions{
		DNSName:     "",
		Roots:       roots,
		CurrentTime: time.Now(),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	})
	assert.NoError(t, err, "cert was not self signed")
}
