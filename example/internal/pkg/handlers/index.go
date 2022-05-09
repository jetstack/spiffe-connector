package handlers

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"gocloud.dev/blob"
)

//go:embed index.html
var indexTemplate string

func NewIndexHandler(gcsBucket, s3Bucket *blob.Bucket) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
}
