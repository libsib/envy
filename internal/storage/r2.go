package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/pradeepbgs/envy/internal/config"
)

type R2 struct {
	client *s3.Client
	bucket string
}

func New(cfg *config.Config) *R2 {
	c := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.R2Endpoint),
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
	})

	return &R2{
		client: c,
		bucket: cfg.Bucket,
	}
}

func (r *R2) Upload(ctx context.Context, key string, data []byte) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (r *R2) Download(ctx context.Context, key string) ([]byte, error) {
	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, environmentError(key, err)
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (r *R2) List(ctx context.Context) ([]string, error) {
	out, err := r.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &r.bucket,
	})
	if err != nil {
		return nil, err
	}

	var keys []string

	for _, obj := range out.Contents {
		if strings.HasSuffix(*obj.Key, ".enc") {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}

func (r *R2) Delete(ctx context.Context, r2_key_name string) error {
	_, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket), Key: aws.String(r2_key_name),
	})
	if err != nil {
		return environmentError(r2_key_name, err)
	}
	_, err = r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(r2_key_name),
	})
	if err != nil {
		return err
	}

	// S3/R2 DeleteObject always returns 204 even for non-existent objects.
	// Verify the object is actually gone.
	_, err = r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(r2_key_name),
	})
	if err == nil {
		return fmt.Errorf("delete request succeeded but object '%s' still exists in bucket '%s'", r2_key_name, r.bucket)
	}
	// Only a not-found response confirms that the object is gone.
	if isNotFound(err) {
		return nil
	}
	return err
}

func isNotFound(err error) bool {
	var apiError smithy.APIError
	return errors.As(err, &apiError) && (apiError.ErrorCode() == "NoSuchKey" || apiError.ErrorCode() == "NotFound")
}

func environmentError(key string, err error) error {
	if isNotFound(err) {
		return fmt.Errorf("env '%s' not found: %w", strings.TrimSuffix(key, ".enc"), err)
	}
	return err
}
