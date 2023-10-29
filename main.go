package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awsSqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/sqs"
	"github.com/numaproj/numaflow-go/pkg/sourcer"
	"log"
	"os"
)

func initSess() *session.Session {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:      aws.String(os.Getenv("REGION")),
			Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY"), os.Getenv("AWS_SECRET"), ""),
			Endpoint:    aws.String(os.Getenv("AWS_END_POINT")),
		},
		SharedConfigState: session.SharedConfigDisable,
	}))
	return sess
}

func main() {
	sqsClient := awsSqs.New(initSess())
	awsSqsSource := sqs.NewAWSSqsSource(sqsClient, os.Getenv("QUEUE_NAME"))
	err := sourcer.NewServer(awsSqsSource).Start(context.Background())
	if err != nil {
		log.Panic("Failed to start source server : ", err)
	}

}
