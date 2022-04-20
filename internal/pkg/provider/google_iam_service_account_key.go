package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"time"

	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func NewGoogleIAMServiceAccountKeyProvider(ctx context.Context, options GoogleIAMServiceAccountKeyProviderOptions) (GoogleIAMServiceAccountKeyProvider, error) {
	// TODO this is from a private package variable, find a way to determine it dynamically
	pingHost := "iam.googleapis.com:https"

	if options.Endpoint != "" {
		ep, err := url.Parse(options.Endpoint)
		if err != nil {
			return GoogleIAMServiceAccountKeyProvider{}, fmt.Errorf("failed to parse supplied endpoint: %w", err)
		}
		if ep.Scheme != "https" && ep.Scheme != "http" {
			return GoogleIAMServiceAccountKeyProvider{}, fmt.Errorf("supplied endpoint value should have http(s) scheme: %q", options.Endpoint)
		}
		if ep.Host == "" {
			return GoogleIAMServiceAccountKeyProvider{}, fmt.Errorf("supplied endpoint value should have host set")
		}
		if ep.Path != "" {
			return GoogleIAMServiceAccountKeyProvider{}, fmt.Errorf("supplied endpoint value should not have path set")
		}

		pingHost = ep.Host

		if ep.Port() != "" {
			pingHost := fmt.Sprintf("%s:http", pingHost)
			if ep.Scheme == "https" {
				pingHost = fmt.Sprintf("%s:https", pingHost)
			}
		}

		options.ClientOptions = append(options.ClientOptions, option.WithEndpoint(options.Endpoint))
	}

	service, err := iam.NewService(ctx, options.ClientOptions...)
	if err != nil {
		return GoogleIAMServiceAccountKeyProvider{}, fmt.Errorf("failed to create IAM service: %w", err)
	}

	return GoogleIAMServiceAccountKeyProvider{
		iamService: service,
		pingHost:   pingHost,
	}, nil
}

type GoogleIAMServiceAccountKeyProviderOptions struct {
	// Endpoint is passed to the service client as withEndpoint but also used for the ping hostname
	Endpoint string
	// ClientOptions are GCP service client options which are used to initialize the nested GCP IAM service client
	ClientOptions []option.ClientOption
}

type GoogleIAMServiceAccountKeyProvider struct {
	iamService *iam.Service
	pingHost   string
}

func (p *GoogleIAMServiceAccountKeyProvider) Name() string {
	return "GoogleIAMServiceAccountKeyProvider"
}

func (p *GoogleIAMServiceAccountKeyProvider) Ping() error {
	_, err := net.DialTimeout("tcp", p.pingHost, time.Second*3)

	if err != nil {

		return fmt.Errorf("provider ping failed: %w", err)
	}

	return nil
}

func (p *GoogleIAMServiceAccountKeyProvider) GetCredential(objectReference string) (Credential, error) {
	// - for the project will infer it from the objectReference
	resource := "projects/-/serviceAccounts/" + objectReference
	request := &iam.CreateServiceAccountKeyRequest{}
	key, err := p.iamService.Projects.ServiceAccounts.Keys.Create(resource, request).Do()
	fmt.Printf("%T", err)
	if err != nil {
		return Credential{}, fmt.Errorf("failed to create service account key: %w", err)
	}

	jsonKeyFile, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return Credential{}, fmt.Errorf("failed to create service account key file JSON: %w", err)
	}

	return Credential{
		Files: []CredentialFile{
			{
				Path:     "key.json",
				Mode:     0644,
				Contents: []byte(jsonKeyFile),
			},
		},
	}, nil
}
