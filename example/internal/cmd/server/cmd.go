package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
	"gocloud.dev/blob"

	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"

	"github.com/jetstack/spiffe-connector-vault/example/internal/pkg/handlers"
)

// s3Bucket is a handle to a bucket in S3 which is
// initialized at start up and used in handlers
var s3Bucket *blob.Bucket

// gcsBucket is a handle to a bucket in GCS which is
// initialized at start up and used in handlers
var gcsBucket *blob.Bucket

func Run(ctx *cli.Context) error {
	var err error
	// load the app config
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/config/")
	if err = viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config.yaml file not found")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home dir")
	}
	log.Println("using homedir:", home)

	// configure backoff
	backOffConfig := backoff.NewExponentialBackOff()
	backOffConfig.MaxInterval, err = time.ParseDuration("3s")

	// wait for aws creds
	err = backoff.Retry(func() error {
		var err error

		s3Bucket, err = blob.OpenBucket(context.Background(), fmt.Sprintf("s3://%s?region=%s", viper.GetString("aws.bucketName"), viper.GetString("aws.region")))
		if err != nil {
			log.Println(fmt.Errorf("waiting for aws credentials (%w)", err))
			return err
		}

		accessible, err := s3Bucket.IsAccessible(context.Background())
		if err != nil {
			log.Println(fmt.Errorf("waiting waiting to test aws credentials (%w)", err))
			return err
		}

		if !accessible {
			return backoff.Permanent(fmt.Errorf("s3 bucket was not accessible with credentials"))
		}
		return nil
	}, backOffConfig)
	if err != nil {
		return fmt.Errorf("failed to get aws credentials after back off: %w", err)
	}
	log.Println("aws credentials present")
	defer s3Bucket.Close()

	// wait for gcp creds
	err = backoff.Retry(func() error {
		var err error

		// we check if file exists here since the blob.OpenBucket seems to
		// cache the fact that the credentials are not present, and continues
		// to fail even if present
		adcPath := "~/.config/gcloud/application_default_credentials.json"
		if s := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); s != "" {
			adcPath = s
		}
		path, err := homedir.Expand(adcPath)
		if err != nil {
			return backoff.Permanent(fmt.Errorf("failed to build path for application_default_credentials: %w", err))
		}

		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			log.Println(err)
			log.Println("waiting for gcp credentials file")
			return err
		}

		gcsBucket, err = blob.OpenBucket(context.Background(), fmt.Sprintf("gs://%s", viper.GetString("gcp.bucketName")))
		if err != nil {
			log.Println(err)
			log.Println("waiting for gcp bucket handle")
			return err
		}

		accessible, err := gcsBucket.IsAccessible(context.Background())
		if err != nil || !accessible {
			return backoff.Permanent(fmt.Errorf("gcs bucket was not accessible with credentials"))
		}
		return nil
	}, backOffConfig)
	if err != nil {
		return fmt.Errorf("failed to get gcp credentials after back off: %w", err)
	}
	log.Println("gcp credentials present")
	defer gcsBucket.Close()

	// start the server
	http.HandleFunc("/styles.css", handlers.StylesHandler)
	http.HandleFunc("/move", handlers.NewMoveHandler(gcsBucket, s3Bucket))
	http.HandleFunc("/", handlers.NewIndexHandler(gcsBucket, s3Bucket))
	port := "3000"
	if p := viper.GetString("http.port"); p != "" {
		port = p
	}
	log.Printf("serving on port %s", port)

	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		return fmt.Errorf("running server failed with error: %w", err)
	}
	return nil
}
