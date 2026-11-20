package secrets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// AWSProvider reads secrets from AWS Secrets Manager. With an endpoint set it talks to
// an S3-style compatible service instead, such as LocalStack for local testing.
type AWSProvider struct {
	client *secretsmanager.Client
}

// NewAWSProvider builds a Secrets Manager client from the standard AWS configuration
// chain (environment, shared config, instance role). endpoint may be empty for AWS.
func NewAWSProvider(ctx context.Context, region, endpoint string) (*AWSProvider, error) {
	if region == "" {
		return nil, fmt.Errorf("aws secret provider needs a region")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	client := secretsmanager.NewFromConfig(cfg, func(o *secretsmanager.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
	return &AWSProvider{client: client}, nil
}

// providerCallTimeout bounds one call to the secret service. The SDK retries a refused
// connection with backoff, and a failing lookup must not stall startup for minutes.
var providerCallTimeout = 10 * time.Second

// Get returns the string value of the secret at path.
func (p *AWSProvider) Get(ctx context.Context, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, providerCallTimeout)
	defer cancel()
	out, err := p.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(path)})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return "", fmt.Errorf("%w: %s", ErrSecretNotFound, path)
		}
		return "", fmt.Errorf("%w: get %s: %v", ErrProviderUnavailable, path, err)
	}
	if out.SecretString == nil {
		return "", fmt.Errorf("secret %s is binary; only string secrets are supported", path)
	}
	return *out.SecretString, nil
}

// List returns the names of the secrets whose names start with prefix.
func (p *AWSProvider) List(ctx context.Context, prefix string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, providerCallTimeout)
	defer cancel()
	var names []string
	var token *string
	for {
		out, err := p.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
			Filters:   []types.Filter{{Key: types.FilterNameStringTypeName, Values: []string{prefix}}},
			NextToken: token,
		})
		if err != nil {
			return nil, fmt.Errorf("%w: list %s: %v", ErrProviderUnavailable, prefix, err)
		}
		for _, s := range out.SecretList {
			if s.Name != nil {
				names = append(names, *s.Name)
			}
		}
		if out.NextToken == nil {
			return names, nil
		}
		token = out.NextToken
	}
}
