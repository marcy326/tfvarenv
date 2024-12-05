package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func GetAWSAccountID(region string) (string, error) {
	// Load default AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region), // 必要に応じてデフォルトリージョンを設定
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
					SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
					SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
				}, nil
			},
		))),
	)
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

func DownloadFromS3(s3Key, localFile, region string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg)

	// Parse the S3 key to get bucket and key
	bucket, key, err := parseS3Key(s3Key)
	if err != nil {
		return err
	}

	// Get the object from S3
	resp, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("unable to download item %q, %v", s3Key, err)
	}
	defer resp.Body.Close()

	// Create the local file
	outFile, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("failed to create file %q, %v", localFile, err)
	}
	defer outFile.Close()

	// Write the contents to the file
	_, err = outFile.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file %q, %v", localFile, err)
	}

	return nil
}

func UploadToS3(localFile, s3Key, region string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg)

	// Parse the S3 key to get bucket and key
	bucket, key, err := parseS3Key(s3Key)
	if err != nil {
		return err
	}

	// Open the local file
	file, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", localFile, err)
	}
	defer file.Close()

	// Upload the file to S3
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("unable to upload %q to %q, %v", localFile, s3Key, err)
	}

	return nil
}

func parseS3Key(s3Key string) (string, string, error) {
	// Assume s3Key is in the format "s3://bucket/key"
	if len(s3Key) < 5 || s3Key[:5] != "s3://" {
		return "", "", fmt.Errorf("invalid S3 key format: %s", s3Key)
	}

	parts := strings.SplitN(s3Key[5:], "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid S3 key format: %s", s3Key)
	}

	return parts[0], parts[1], nil
}
