package filestore

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Storage implements FileStorage for S3-compatible services (AWS, DigitalOcean Spaces, MinIO)
type S3Storage struct {
	client     *s3.Client
	tranfer    *transfermanager.Client
	bucketName string
	region     string
	endpoint   string
}

// NewS3Storage initializes a new S3 client
func NewS3Storage(accessKey, secretKey, endpoint, region, bucket string) (*S3Storage, error) {
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(creds),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = endpoint != ""
	})

	transferClient := transfermanager.New(client)

	return &S3Storage{
		client:     client,
		tranfer:    transferClient,
		bucketName: bucket,
		region:     region,
		endpoint:   endpoint,
	}, nil
}

// Save uploads a file to S3 bucket
func (s *S3Storage) Save(ctx context.Context, file io.Reader, path string) (int64, error) {
	object, err := s.tranfer.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
		Body:   file,
		ACL:    "private",
	})
	if err != nil {
		return 0, fmt.Errorf("failed to save file to S3: %w", err)
	}

	return *object.ContentLength, nil
}

// Get retrieves a file stream from S3
func (s *S3Storage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file from s3: %w", err)
	}

	return output.Body, nil
}

// Delete removes files from storage
func (s *S3Storage) Delete(ctx context.Context, paths []string) (successCount, failureCount int, err error) {
	var objectIds []types.ObjectIdentifier
	for _, path := range paths {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(path)})
	}

	output, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucketName),
		Delete: &types.Delete{Objects: objectIds, Quiet: aws.Bool(true)},
	})
	if err != nil {
		return 0, 0, err
	}

	successCount = len(output.Deleted)
	failureCount = len(output.Errors)

	return successCount, failureCount, nil
}
