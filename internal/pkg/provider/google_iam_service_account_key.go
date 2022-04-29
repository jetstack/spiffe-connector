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
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
)

func NewGoogleIAMServiceAccountKeyProvider(ctx context.Context, options GoogleIAMServiceAccountKeyProviderOptions) (GoogleIAMServiceAccountKeyProvider, error) {
	// This is the default host used to check the functioning of the provider
	// TODO this is from a private package variable, find a way to determine it dynamically
	pingHost := "iam.googleapis.com:https"

	// if the options endpoint has been set then we need to check it's valid and update the pingHost value to make sure
	// that Ping functions correctly and the Google client is initialised with the new host
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

		// pingHost is set to the Host of the validated URL, this will contain a port if one was set
		pingHost = ep.Host
		// if there is not a port set in the supplied endpoint, then we get the net package to dial on http or https
		// based on the scheme
		if ep.Port() == "" {
			pingHost = fmt.Sprintf("%s:http", ep.Host)
			if ep.Scheme == "https" {
				pingHost = fmt.Sprintf("%s:https", ep.Host)
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

func (p *GoogleIAMServiceAccountKeyProvider) GetCredential(objectReference string) (*proto.Credential, error) {
	// - for the project will infer it from the objectReference
	resource := "projects/-/serviceAccounts/" + objectReference
	request := &iam.CreateServiceAccountKeyRequest{}
	key, err := p.iamService.Projects.ServiceAccounts.Keys.Create(resource, request).Do()
	if err != nil {
		return &proto.Credential{}, fmt.Errorf("failed to create service account key: %w", err)
	}

	jsonKeyFile, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
	if err != nil {
		return &proto.Credential{}, fmt.Errorf("failed to create service account key file JSON: %w", err)
	}

	notAfter, err := time.Parse(time.RFC3339, key.ValidBeforeTime)
	if err != nil {
		return &proto.Credential{}, fmt.Errorf("failed to parse credential valid before time: %w", err)
	}

	return &proto.Credential{
		NotAfter: timestamppb.New(notAfter),
		Files: []*proto.File{
			{
				Path:     "key.json",
				Mode:     0644,
				Contents: []byte(jsonKeyFile),
			},
		},
	}, nil
}
