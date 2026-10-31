package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func testR2(t *testing.T, handler http.HandlerFunc) *R2 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &R2{bucket: "envs", client: s3.New(s3.Options{
		BaseEndpoint: aws.String(server.URL), Region: "auto", UsePathStyle: true,
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""), RetryMaxAttempts: 1,
	})}
}

func TestDownloadMissingEnvironment(t *testing.T) {
	client := testR2(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/xml")
		writer.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(writer, "<Error><Code>NoSuchKey</Code><Message>The key does not exist</Message></Error>")
	})
	_, err := client.Download(context.Background(), "staging.enc")
	if err == nil || !strings.Contains(err.Error(), "env 'staging' not found") {
		t.Fatalf("Download error = %v", err)
	}
}

func TestDeleteMissingEnvironment(t *testing.T) {
	deletions := 0
	client := testR2(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodDelete {
			deletions++
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		writer.WriteHeader(http.StatusNotFound)
	})
	err := client.Delete(context.Background(), "staging.enc")
	if err == nil || !strings.Contains(err.Error(), "env 'staging' not found") {
		t.Fatalf("Delete error = %v", err)
	}
	if deletions != 0 {
		t.Errorf("made %d delete requests for a missing environment", deletions)
	}
}

func TestDeletePropagatesVerificationFailure(t *testing.T) {
	deleted := false
	client := testR2(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodDelete {
			deleted = true
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if deleted {
			writer.WriteHeader(http.StatusForbidden)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
	if err := client.Delete(context.Background(), "staging.enc"); err == nil {
		t.Fatal("Delete hid the verification failure")
	}
}

func TestDeleteExistingEnvironment(t *testing.T) {
	deleted := false
	client := testR2(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodDelete {
			deleted = true
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if deleted {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
	if err := client.Delete(context.Background(), "staging.enc"); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadMissingEnvironmentWithoutErrorBody(t *testing.T) {
	client := testR2(t, func(writer http.ResponseWriter, request *http.Request) { writer.WriteHeader(http.StatusNotFound) })
	_, err := client.Download(context.Background(), "staging.enc")
	if err == nil || !strings.Contains(err.Error(), "env 'staging' not found") {
		t.Fatalf("Download error = %v", err)
	}
}

func TestDownloadMissingBucketIsNotMissingEnvironment(t *testing.T) {
	client := testR2(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/xml")
		writer.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(writer, "<Error><Code>NoSuchBucket</Code><Message>Bucket does not exist</Message></Error>")
	})
	_, err := client.Download(context.Background(), "staging.enc")
	if err == nil || strings.Contains(err.Error(), "env 'staging' not found") {
		t.Fatalf("Download error = %v", err)
	}
}
