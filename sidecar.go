package s3sidecar

import (
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Sidecar struct {
	interval time.Duration
	sess     *session.Session
	bucket   string
	key      string
	//working Directory
	wDirectory string
	//updates Directory
	uDirectory string
}

func NewS3Sidecar(_interval time.Duration, region, bucket, key, workingDirectory, updatesDirectory string) (*S3Sidecar, error) {
	// Initialize a session in us-west-2 that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials.
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return nil, err
	}
	return &S3Sidecar{_interval, sess, bucket, key, workingDirectory, updatesDirectory}, nil
}

func (s *S3Sidecar) downloadFile() error {
	file, err := os.Create(s.wDirectory + "/" + s.key)
	if err != nil {
		return err
	}
	downloader := s3manager.NewDownloader(s.sess)
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(s.key),
		})
	log.Println("File downloaded from S3")
	return err
}

func (s *S3Sidecar) uploadFile() error {
	file, err := os.Open(s.uDirectory + "/" + s.key)
	if err != nil {
		return err
	}
	uploader := s3manager.NewUploader(s.sess)
	_, err = uploader.Upload(
		&s3manager.UploadInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(s.key),
			Body:   file,
		})
	log.Println("File uploaded to S3")
	return err
}

func (s *S3Sidecar) hasChanges(lastLocal time.Time) (bool, *time.Time, error) {
	svc := s3.New(s.sess, &aws.Config{
		DisableRestProtocolURICleaning: aws.Bool(true),
	})
	out, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return false, nil, err
	}
	for _, object := range out.Contents {
		if *object.Key == s.key && lastLocal.Before(*object.LastModified) {
			return true, object.LastModified, nil
		}
	}
	return false, nil, nil
}

func (s *S3Sidecar) localUpdate(lastLocal time.Time) (bool, error) {
	fileInfo, err := os.Stat(s.uDirectory + "/" + s.key)
	if err == nil {
		if lastLocal.Before(fileInfo.ModTime()) {
			return true, nil
		}
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *S3Sidecar) Start(done <-chan interface{}) <-chan error {
	errors := make(chan error)
	ticker := time.NewTicker(s.interval * time.Second)
	// 1970-01-01 00:00:00 +0000 UTC
	lastLocal := time.Unix(0, 0)
	go func() {
		for {
			select {
			case <-ticker.C:
				pushFile, err := s.localUpdate(lastLocal)
				if err != nil {
					errors <- err
					break
				}
				if pushFile {
					err := s.uploadFile()
					if err != nil {
						errors <- err
						break
					}
				}
				update, lastRemote, err := s.hasChanges(lastLocal)
				if err != nil {
					errors <- err
					break
				}
				if update {
					err := s.downloadFile()
					if err != nil {
						errors <- err
						break
					}
					lastLocal = *lastRemote
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return errors
}
