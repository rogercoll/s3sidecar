package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/rogercoll/s3sidecar"
)

func main() {
	interval, ok := os.LookupEnv("INTERVAL")
	if !ok {
		log.Fatal("INTERVAL is not present")
	}
	region, ok := os.LookupEnv("AWS_REGION")
	if !ok {
		log.Fatal("AWS_REGION is not present")
	}
	bucket, ok := os.LookupEnv("S3_BUCKET")
	if !ok {
		log.Fatal("S3_BUCKET is not present")
	}
	object, ok := os.LookupEnv("S3_OBJECT")
	if !ok {
		log.Fatal("S3_OBJECT is not present")
	}
	wdir, ok := os.LookupEnv("W_DIR")
	if !ok {
		log.Fatal("W_DIR is not present")
	}
	udir, ok := os.LookupEnv("U_DIR")
	if !ok {
		log.Fatal("U_DIR is not present")
	}
	iinterval, err := strconv.Atoi(interval)
	if err != nil {
		log.Fatal(err)
	}
	s, err := s3sidecar.NewS3Sidecar(time.Duration(int64(iinterval)), region, bucket, object, wdir, udir)
	if err != nil {
		log.Fatal(err)
	}
	done := make(chan interface{})
	errs := s.Start(done)
	totalErrs := 0
	for {
		select {
		case err := <-errs:
			totalErrs += 1
			log.Println(err)
			if totalErrs > 5 {
				log.Println("More than 5 errors returned, exiting the program...")
				close(done)
				os.Exit(1)
			}

		}
	}
}
