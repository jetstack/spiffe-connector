package main

import (
	"context"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"gocloud.dev/blob"

	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

// s3Bucket is a handle to a bucket in S3 which is
// initialized at start up and used in handlers
var s3Bucket *blob.Bucket

// gcsBucket is a handle to a bucket in GCS which is
// initialized at start up and used in handlers
var gcsBucket *blob.Bucket

//go:embed index.html
var indexTemplate string

//go:embed css/*
var cssContent embed.FS

func stylesHandler(w http.ResponseWriter, r *http.Request) {
	normalizeData, err := cssContent.ReadFile("css/normalize.min.css")
	if err != nil {
		err = fmt.Errorf("failed to load normalize css content from storage: %w", err)
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	tachyonsData, err := cssContent.ReadFile("css/tachyons.min.css")
	if err != nil {
		err = fmt.Errorf("failed to load tachyons css content from storage: %w", err)
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	siteStyleData, err := cssContent.ReadFile("css/styles.css")
	if err != nil {
		err = fmt.Errorf("failed to load app styles css content from storage: %w", err)
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	allCSSData := ""
	for _, b := range []*[]byte{&normalizeData, &tachyonsData, &siteStyleData} {
		allCSSData += string(*b) + "\n"
	}

	w.Header().Set("Content-Type", "text/css")
	fmt.Fprint(w, allCSSData)
}

// moveHandler processes a 'swap operation' where files are moved from one bucket to another
// depending on where they are found. If the files are in one bucket, they are moved to the
// other and vice versa.
func moveHandler(w http.ResponseWriter, r *http.Request) {
	var s3BucketFiles []string
	iter := s3Bucket.List(nil)
	for {
		obj, err := iter.Next(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			err = fmt.Errorf("failed to list items in s3 bucket: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		s3BucketFiles = append(s3BucketFiles, obj.Key)
	}

	var gcpBucketFiles []string
	iter = gcsBucket.List(nil)
	for {
		obj, err := iter.Next(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			err = fmt.Errorf("failed to list items in gcp bucket: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		gcpBucketFiles = append(gcpBucketFiles, obj.Key)
	}

	sourceBucket := s3Bucket
	destinationBucket := gcsBucket
	sourceFiles := s3BucketFiles
	// determine which directory the file is in, and move from there to the destination
	if len(gcpBucketFiles) > len(s3BucketFiles) {
		sourceBucket = gcsBucket
		destinationBucket = s3Bucket
		sourceFiles = gcpBucketFiles
	}

	for _, file := range sourceFiles {
		sourceReader, err := sourceBucket.NewReader(context.Background(), file, nil)
		if err != nil {
			err = fmt.Errorf("failed to get handle for object in source bucket: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		destinationWriter, err := destinationBucket.NewWriter(context.Background(), file, nil)
		if err != nil {
			err = fmt.Errorf("failed to get handle for object in destination bucket: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		_, err = io.Copy(destinationWriter, sourceReader)
		if err != nil {
			err = fmt.Errorf("failed to copy file: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		err = sourceReader.Close()
		if err != nil {
			err = fmt.Errorf("failed to close source reader: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		err = destinationWriter.Close()
		if err != nil {
			err = fmt.Errorf("failed to close destination writer: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		err = sourceBucket.Delete(context.Background(), file)
		if err != nil {
			err = fmt.Errorf("failed to remove the old object from source: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var s3BucketFiles []string
	iter := s3Bucket.List(nil)
	for {
		obj, err := iter.Next(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			err = fmt.Errorf("failed to list items in s3 bucket: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		s3BucketFiles = append(s3BucketFiles, obj.Key)
	}

	var gcpBucketFiles []string
	iter = gcsBucket.List(nil)
	for {
		obj, err := iter.Next(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			err = fmt.Errorf("failed to list items in gcs bucket: %w", err)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		gcpBucketFiles = append(gcpBucketFiles, obj.Key)
	}

	tmplt := template.New("index")
	tmplt, err := tmplt.Parse(indexTemplate)
	if err != nil {
		err = fmt.Errorf("failed to parse template: %w", err)
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	p := struct {
		S3Files  []string
		GCSFiles []string
	}{
		S3Files:  s3BucketFiles,
		GCSFiles: gcpBucketFiles,
	}

	tmplt.Execute(w, p)
}

func main() {
	var err error
	// load the app config
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/config/")
	if err = viper.ReadInConfig(); err != nil {
		log.Fatal("config.yaml file not found")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("failed to determine home dir")
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
		log.Fatal(err)
		return
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
		log.Fatal(err)
		return
	}
	log.Println("gcp credentials present")
	defer gcsBucket.Close()

	// start the server
	http.HandleFunc("/styles.css", stylesHandler)
	http.HandleFunc("/move", moveHandler)
	http.HandleFunc("/", indexHandler)
	port := "3000"
	if p := viper.GetString("http.port"); p != "" {
		port = p
	}
	log.Printf("serving on port %s", port)

	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
