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
}

func NewS3Sidecar(_interval time.Duration, region, bucket, key, workingDirectory string) (*S3Sidecar, error) {
	// Initialize a session in us-west-2 that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials.
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return nil, err
	}
	return &S3Sidecar{_interval, sess, bucket, key, workingDirectory}, nil
}

func (s *S3Sidecar) downloadFile(remoteTime *time.Time) error {
	filename := s.wDirectory + "/" + s.key
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	downloader := s3manager.NewDownloader(s.sess)
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(s.key),
		})
	err = os.Chtimes(filename, *remoteTime, *remoteTime)
	if err != nil {
		return err
	}
	log.Println("File downloaded from S3")
	return err
}

func (s *S3Sidecar) uploadFile() error {
	filename := s.wDirectory + "/" + s.key
	file, err := os.Open(s.wDirectory + "/" + s.key)
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
	lastRemote, err := s.lastRemote()
	if err != nil {
		return err
	}
	err = os.Chtimes(filename, *lastRemote, *lastRemote)
	if err != nil {
		return err
	}
	log.Println("File uploaded to S3")
	return err
}

func (s *S3Sidecar) lastState() time.Time {
	fileInfo, err := os.Stat(s.wDirectory + "/" + s.key)
	if err == nil {
		return fileInfo.ModTime()
	}
	return time.Unix(0, 0)
}

func (s *S3Sidecar) lastRemote() (*time.Time, error) {
	svc := s3.New(s.sess, &aws.Config{
		DisableRestProtocolURICleaning: aws.Bool(true),
	})
	out, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return nil, err
	}
	for _, object := range out.Contents {
		if *object.Key == s.key {
			return object.LastModified, nil
		}
	}
	t := time.Unix(0, 0)
	return &t, nil
}

func (s *S3Sidecar) Start(done <-chan interface{}) <-chan error {
	errors := make(chan error)
	ticker := time.NewTicker(s.interval * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				lastLocal := s.lastState()
				lastRemote, err := s.lastRemote()
				if err != nil {
					errors <- err
					break
				}
				if lastLocal.Before(*lastRemote) {
					err := s.downloadFile(lastRemote)
					if err != nil {
						errors <- err
						break
					}
				} else if lastLocal.After(*lastRemote) {
					err := s.uploadFile()
					if err != nil {
						errors <- err
						break
					}
				} else {
					log.Println("File already up-to-date")
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return errors
}
