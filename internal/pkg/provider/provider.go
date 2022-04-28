// Package provider represents existing credentials that you can swap your SPIFFE ID for.
package provider

import (
	"io/fs"
	"time"
)

type Provider interface {
	Name() string
	Ping() error
	GetCredential(objectReference string) (*Credential, error)
}

type Credential struct {
	Files    []CredentialFile
	EnvVars  map[string]string
	Username *string
	Password *string
	Token    *string
	NotAfter time.Time
}

type CredentialFile struct {
	Path     string
	Mode     fs.FileMode
	Contents []byte
}
