package sqs

import (
	"context"
	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/config"
	"github.com/numaproj/numaflow-go/pkg/sourcer"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
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

func TestAWSSqsSource_Read(t *testing.T) {
	awsConfig := config.AwsConfig{
		AwsRegion:    "us-east-1",
		AwsAccessKey: "AKIAS2LNCO4EY3JOZMTY",
		AwsSecret:    "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",
	}
	sqsSource := NewAWSSqsSource(awsConfig)
	assert.NotNil(t, sqsSource.sqsServiceClient)
	rr := &ReadRequest{}
	msgChan := make(chan sourcer.Message)
	sqsSource.Read(context.Background(), rr, msgChan)
	for ms := range msgChan {
		log.Print(ms)
	}

}

func TestAWSSqsSource_Pending(t *testing.T) {
	awsConfig := config.AwsConfig{
		AwsRegion:    "us-east-1",
		AwsAccessKey: "AKIAS2LNCO4EY3JOZMTY",
		AwsSecret:    "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",
	}
	sqsSource := NewAWSSqsSource(awsConfig)
	assert.NotNil(t, sqsSource.sqsServiceClient)
	sqsSource.Pending(context.TODO())

}
