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

func SelfSignedServingCert() (tls.Certificate, error) {
	tlsCert := tls.Certificate{}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tlsCert.PrivateKey = key

	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return tls.Certificate{}, err
	}
	subj := pkix.Name{
		Country:            []string{"GB"},
		Organization:       []string{"Jetstack"},
		OrganizationalUnit: []string{"Product"},
		SerialNumber:       serial.String(),
	}
	uri, err := url.Parse("spiffe://dummy.domain/spiffe-connector")
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		PublicKeyAlgorithm: x509.ECDSA,
		PublicKey:          key.Public(),
		SerialNumber:       serial,
		Issuer:             subj,
		Subject:            subj,
		NotBefore:          time.Now(),
		NotAfter:           time.Now().Add(100 * time.Hour * 24 * 365),
		KeyUsage:           x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		IsCA:               true,
		DNSNames:           nil,
		EmailAddresses:     nil,
		IPAddresses:        nil,
		URIs:               []*url.URL{uri},
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		return tls.Certificate{}, err
	}
	tlsCert.Certificate = [][]byte{cert}

	return tlsCert, nil
}
