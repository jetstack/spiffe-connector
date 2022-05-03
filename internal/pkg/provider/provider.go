// Package provider represents existing credentials that you can swap your SPIFFE ID for.
package provider

import (
	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
)

type Provider interface {
	Name() string
	Ping() error
	GetCredential(objectReference string) (*proto.Credential, error)
}
