package aws

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type client struct {
	cfg       aws.Config
	s3Client  *s3.Client
	stsClient *sts.Client
}

// NewClient creates a new AWS client
func NewClient(region string) (Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
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
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	return &client{
		cfg:       cfg,
		s3Client:  s3.NewFromConfig(cfg),
		stsClient: sts.NewFromConfig(cfg),
	}, nil
}

func (c *client) GetAccountID(ctx context.Context) (string, error) {
	output, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity: %w", err)
	}
	return *output.Account, nil
}

func (c *client) CheckBucketVersioning(ctx context.Context, bucket string) error {
	output, err := c.s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to check bucket versioning: %w", err)
	}

	if output.Status != types.BucketVersioningStatusEnabled {
		return fmt.Errorf("versioning is not enabled on bucket %s", bucket)
	}
	return nil
}

func (c *client) UploadFile(ctx context.Context, input *UploadInput) (*UploadOutput, error) {
	result, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(input.Bucket),
		Key:         aws.String(input.Key),
		Body:        bytes.NewReader(input.Content),
		ContentType: aws.String("application/x-tfvars"),
		Metadata:    input.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &UploadOutput{
		VersionID: *result.VersionId,
		ETag:      *result.ETag,
	}, nil
}

func (c *client) DownloadFile(ctx context.Context, input *DownloadInput) (*DownloadOutput, error) {
	getObjInput := &s3.GetObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
	}
	if input.VersionID != "" {
		getObjInput.VersionId = aws.String(input.VersionID)
	}

	result, err := c.s3Client.GetObject(ctx, getObjInput)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer result.Body.Close()

	content, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return &DownloadOutput{
		Content:     content,
		VersionID:   *result.VersionId,
		Metadata:    result.Metadata,
		ContentType: *result.ContentType,
	}, nil
}

func (c *client) ListVersions(ctx context.Context, input *ListVersionsInput) (*ListVersionsOutput, error) {
	result, err := c.s3Client.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{
		Bucket:  aws.String(input.Bucket),
		Prefix:  aws.String(input.Key),
		MaxKeys: aws.Int32(input.MaxKeys),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	versions := make([]VersionInfo, 0, len(result.Versions))
	for _, v := range result.Versions {
		obj, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket:    aws.String(input.Bucket),
			Key:       aws.String(input.Key),
			VersionId: v.VersionId,
		})
		if err != nil {
			continue
		}

		versions = append(versions, VersionInfo{
			VersionID:   *v.VersionId,
			Hash:        obj.Metadata["Hash"],
			Timestamp:   *v.LastModified,
			Description: obj.Metadata["Description"],
			Size:        *v.Size,
			IsLatest:    *v.IsLatest,
			Metadata:    obj.Metadata,
		})
	}

	return &ListVersionsOutput{
		Versions:    versions,
		IsTruncated: *result.IsTruncated,
		NextMarker:  aws.ToString(result.NextKeyMarker),
	}, nil
}
