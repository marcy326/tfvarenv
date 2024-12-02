package utils

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func GetAWSAccountID() (string, error) {
	// Load default AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("unable to load AWS SDK config, %v", err)
	}

	// Create STS client
	stsClient := sts.NewFromConfig(cfg)

	// Call GetCallerIdentity
	output, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity, %v", err)
	}

	return *output.Account, nil
}
