package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/bootstrap"
)

func main() {
	h, err := bootstrap.NewHandler()

	if err != nil {
		log.Fatal(err)
	}

	lambda.Start(h.Handle)
}
