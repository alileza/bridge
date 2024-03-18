package s3storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Storage struct {
	BucketName string
	Region     string

	routes sync.Map
	svc    *s3.S3
}

func NewS3Storage(bucketName, region string) (*S3Storage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("s3storage: error creating AWS session: %w", err)
	}

	svc := s3.New(sess)

	s3Storage := &S3Storage{
		BucketName: bucketName,
		Region:     region,
		svc:        svc,
	}

	// Load existing data from S3 bucket if it exists
	data, err := s3Storage.downloadFromS3()
	if err != nil {
		log.Printf("s3storage: error downloading data from S3: %v", err)
		return nil, err
	}

	tmp := make(map[string]string)
	if err := json.Unmarshal(data, &tmp); err != nil {
		log.Printf("s3storage: error decoding data from S3: %v", err)
		return nil, err
	}

	for k, v := range tmp {
		s3Storage.routes.Store(k, v)
	}

	return s3Storage, nil
}

func (s3s *S3Storage) Store(key, value interface{}) {
	s3s.routes.Store(key, value)
	if err := s3s.saveToS3(); err != nil {
		log.Printf("s3storage: error saving data to S3: %v", err)
	}
}

func (s3s *S3Storage) Load(key interface{}) (value interface{}, ok bool) {
	return s3s.routes.Load(key)
}

func (s3s *S3Storage) Delete(key interface{}) {
	s3s.routes.Delete(key)
	if err := s3s.saveToS3(); err != nil {
		log.Printf("s3storage: error saving data to S3: %v", err)
	}
}

func (s3s *S3Storage) Range(f func(key, value interface{}) bool) {
	s3s.routes.Range(f)
}

func (s3s *S3Storage) downloadFromS3() ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s3s.BucketName),
		Key:    aws.String("data.json"),
	}

	result, err := s3s.svc.GetObject(input)
	if err != nil {
		return nil, fmt.Errorf("s3storage: error downloading from S3: %w", err)
	}
	defer result.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(result.Body)
	if err != nil {
		return nil, fmt.Errorf("s3storage: error reading data from S3: %w", err)
	}

	return buf.Bytes(), nil
}

func (s3s *S3Storage) saveToS3() error {
	data, err := json.Marshal(s3s.routes)
	if err != nil {
		return fmt.Errorf("s3storage: error encoding data: %w", err)
	}

	input := &s3.PutObjectInput{
		Body:   bytes.NewReader(data),
		Bucket: aws.String(s3s.BucketName),
		Key:    aws.String("data.json"),
	}

	_, err = s3s.svc.PutObject(input)
	if err != nil {
		return fmt.Errorf("s3storage: error uploading to S3: %w", err)
	}

	return nil
}
