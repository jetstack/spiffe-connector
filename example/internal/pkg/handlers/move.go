package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"gocloud.dev/blob"
)

// NewMoveHandler builds a handler which processes a 'swap operation' where files are moved
// from one bucket to another depending on where they are found. If the files are in one
// bucket, they are moved to the other and vice versa.
func NewMoveHandler(gcsBucket, s3Bucket *blob.Bucket) http.HandlerFunc {
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
}
