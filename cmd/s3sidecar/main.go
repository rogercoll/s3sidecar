package main

import (
	"log"
	"os"

	"github.com/rogercoll/s3sidecar"
)

func main() {
	s, err := s3sidecar.NewS3Sidecar(5, "eu-west-1", "neckapps", "dictionary.txt", "/data", "/data/upload")
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
