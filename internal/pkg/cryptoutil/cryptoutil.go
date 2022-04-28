// Package cryptoutil contains reusable utility functions
package cryptoutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math"
	"math/big"
	"net/url"
	"time"
)

// GenerateTestCerts generates a root for the trust domain, and a SPIFFE ID signed by that root
// returning a tls.Certificate containing the CA and leaf.
func GenerateTestCerts(spiffeid string) (tls.Certificate, error) {
	tlsCert := tls.Certificate{}

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tlsCert.PrivateKey = leafKey

	caSerial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return tls.Certificate{}, err
	}
	leafSerial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return tls.Certificate{}, err
	}
	caSubj := pkix.Name{
		Country:            []string{"GB"},
		Organization:       []string{"Jetstack"},
		OrganizationalUnit: []string{"Product"},
		SerialNumber:       caSerial.String(),
	}
	leafSubj := pkix.Name{
		Country:            []string{"GB"},
		Organization:       []string{"Jetstack"},
		OrganizationalUnit: []string{"Product"},
		SerialNumber:       leafSerial.String(),
	}
	uri, err := url.Parse(spiffeid)
	if err != nil {
		return tls.Certificate{}, err
	}

	caUri, err := url.Parse("spiffe://" + uri.Host)
	if err != nil {
		return tls.Certificate{}, err
	}

	caTemplate := &x509.Certificate{
		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		PublicKeyAlgorithm:    x509.ECDSA,
		PublicKey:             caKey.Public(),
		SerialNumber:          caSerial,
		Issuer:                caSubj,
		Subject:               caSubj,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(100 * time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		DNSNames:              nil,
		EmailAddresses:        nil,
		IPAddresses:           nil,
		URIs:                  []*url.URL{caUri},
	}

	leafTemplate := &x509.Certificate{
		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		PublicKeyAlgorithm:    x509.ECDSA,
		PublicKey:             leafKey.Public(),
		SerialNumber:          leafSerial,
		Issuer:                caSubj,
		Subject:               leafSubj,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(99 * time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		IsCA:                  false,
		DNSNames:              nil,
		EmailAddresses:        nil,
		IPAddresses:           nil,
		URIs:                  []*url.URL{uri},
	}

	leafCert, err := x509.CreateCertificate(rand.Reader, leafTemplate, caTemplate, leafKey.Public(), caKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	caCert, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, caKey.Public(), caKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	tlsCert.Certificate = [][]byte{leafCert, caCert}

	return tlsCert, nil
}
