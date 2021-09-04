package s3sidecar

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Sidecar struct {
	interval time.Duration
	region   string
	bucket   string
	object   string
	//working Directory
	wDirectory string
}

func NewS3Sidecar(_interval time.Duration, region, bucket, object, workingDirectory string) (*S3Sidecar, error) {
	// Initialize a session in us-west-2 that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials.
	_, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return nil, err
	}
	return &S3Sidecar{_interval, region, bucket, object, workingDirectory}, nil
}

func (s *S3Sidecar) downloadFile(sess *session.Session) error {
	file, err := os.Create(s.wDirectory + s.object)
	if err != nil {
		return err
	}
	downloader := s3manager.NewDownloader(sess)
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(s.object),
		})
	return err
}

func (s *S3Sidecar) hasChanges(sess *session.Session, lastLocal time.Time) (bool, *time.Time, error) {
	svc := s3.New(sess, &aws.Config{
		DisableRestProtocolURICleaning: aws.Bool(true),
	})
	out, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return false, nil, err
	}
	for _, object := range out.Contents {
		if *object.Key == s.object && lastLocal.Before(*object.LastModified) {
			return true, object.LastModified, nil
		}
	}
	return false, nil, nil
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
				sess, err := session.NewSession(&aws.Config{
					Region: aws.String(s.region)},
				)
				if err != nil {
					errors <- err
					break
				}
				update, lastRemote, err := s.hasChanges(sess, lastLocal)
				if err != nil {
					errors <- err
					break
				}
				if update {
					err := s.downloadFile(sess)
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
