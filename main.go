package main

import (
	"context"
	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/config"
	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/sqs"
	"github.com/numaproj/numaflow-go/pkg/sourcer"
	"log"
	"time"
)

type ReadRequest struct {
}

func (r *ReadRequest) TimeOut() time.Duration {
	return 0
}

func (r *ReadRequest) Count() uint64 {
	return 10
}

func ReadMessage() {
	awsConfig := config.AwsConfig{
		AwsRegion:    "us-east-1",
		AwsAccessKey: "AKIAS2LNCO4EY3JOZMTY",
		AwsSecret:    "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",
	}
	sqsSource := sqs.NewAWSSqsSource(awsConfig)
	rr := &ReadRequest{}
	msgChan := make(chan sourcer.Message)
	sqsSource.Read(context.Background(), rr, msgChan)
	for ms := range msgChan {
		log.Print(ms.Value())
	}

}

func main() {

	ReadMessage()

}
