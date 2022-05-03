package provider

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jetstack/spiffe-connector/internal/pkg/server/proto"
)

// AWSSTSAssumeRoleProviderOptions are the options available to configure a AWSSTSAssumeRoleProvider
type AWSSTSAssumeRoleProviderOptions struct {
	// Endpoint is passed to the session to select with AWS endpoint to use, this is optional
	Endpoint string

	// Region will be used if endpoint is set, defaults to us-east-1
	Region string

	// Duration is how long credentials will be valid for, recommended max: 1hr. Durations greater than 1hr might be
	// blocked by organisation settings.
	Duration int64
}

// AWSSTSAssumeRoleProvider is a provider used to get short lived credentials from AWS STS
type AWSSTSAssumeRoleProvider struct {
	pingHost   string
	stsService *sts.STS
	duration   int64
}

// NewAWSSTSAssumeRoleProvider will configure a new AWSSTSAssumeRoleProvider using the supplied options
func NewAWSSTSAssumeRoleProvider(ctx context.Context, options AWSSTSAssumeRoleProviderOptions) (AWSSTSAssumeRoleProvider, error) {
	// from https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html
	pingHost := "sts.amazonaws.com:https"

	var config aws.Config
	if options.Endpoint != "" {
		ep, err := url.Parse(options.Endpoint)
		if err != nil {
			return AWSSTSAssumeRoleProvider{}, fmt.Errorf("failed to parse supplied endpoint: %w", err)
		}
		if ep.Scheme != "https" && ep.Scheme != "http" {
			return AWSSTSAssumeRoleProvider{}, fmt.Errorf("supplied endpoint value should have http(s) scheme: %q", options.Endpoint)
		}
		if ep.Host == "" {
			return AWSSTSAssumeRoleProvider{}, fmt.Errorf("supplied endpoint value should have host set")
		}
		if ep.Path != "" {
			return AWSSTSAssumeRoleProvider{}, fmt.Errorf("supplied endpoint value should not have path set")
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

		if options.Region == "" {
			// non empty region is needed if setting endpoint
			options.Region = "us-east-1"
		}

		config.Endpoint = &options.Endpoint
		config.Region = &options.Region
	}

	sess, err := session.NewSession(&config)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return AWSSTSAssumeRoleProvider{}, fmt.Errorf("failed to create session: %s: %s", aerr.Code(), aerr.Message())
		}
		return AWSSTSAssumeRoleProvider{}, fmt.Errorf("failed to create session: %w", err)
	}

	duration := int64(60 * 60)
	if options.Duration > 0 {
		duration = options.Duration
	}

	return AWSSTSAssumeRoleProvider{
		stsService: sts.New(sess),
		pingHost:   pingHost,
		duration:   duration,
	}, nil
}

// Name returns the name of the provider
func (p *AWSSTSAssumeRoleProvider) Name() string {
	return "AWSSTSAssumeRoleProvider"
}

// Ping tests the configured credential providing endpoint is reachable
// Note: this does not test AWS authn/authz
func (p *AWSSTSAssumeRoleProvider) Ping() error {
	_, err := net.DialTimeout("tcp", p.pingHost, time.Second*3)

	if err != nil {
		return fmt.Errorf("provider ping failed: %w", err)
	}

	return nil
}

// GetCredential will use STS to get a short lived credential for the given objectReference (Role)
// spiffe-connector must be able to AssumeRole for the supplied role for this to work
func (p *AWSSTSAssumeRoleProvider) GetCredential(objectReference string) (*proto.Credential, error) {
	// sessionName is just a label, there can be many sessions with the same name
	sessionName := "spiffe-connector"
	input := &sts.AssumeRoleInput{
		DurationSeconds: &p.duration,
		RoleSessionName: &sessionName,
		RoleArn:         aws.String(objectReference),
	}

	result, err := p.stsService.AssumeRole(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return &proto.Credential{}, fmt.Errorf("failed to get temporary credentials from STS: %s: %s", aerr.Code(), aerr.Message())
		}
		return &proto.Credential{}, fmt.Errorf("failed to get temporary credentials from STS: %w", err)
	}

	credentialsFile := fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
aws_session_token = %s
`,
		*result.Credentials.AccessKeyId,
		*result.Credentials.SecretAccessKey,
		*result.Credentials.SessionToken,
	)

	return &proto.Credential{
		NotAfter: timestamppb.New(*result.Credentials.Expiration),
		Files: []*proto.File{
			{
				Path:     "~/.aws/credentials",
				Mode:     0644,
				Contents: []byte(credentialsFile),
			},
		},
	}, nil
}
