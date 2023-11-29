package main

/*
Copyright 2022 The Numaproj Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsSqs "github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/numaproj/numaflow-go/pkg/sourcer"

	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/sqs"
)

const (
	awsEndpointURL = "AWS_ENDPOINT_URL"
	sqsQueueName   = "AWS_SQS_QUEUE_NAME"
)

func initClient(ctx context.Context, awsEndpoint string) (*awsSqs.Client, error) {
	// Load default configs for aws based on env variable provided based on
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed loading aws config, err: %v", err)
	}
	var client *awsSqs.Client
	if awsEndpoint != "" {
		client = awsSqs.NewFromConfig(cfg, func(options *awsSqs.Options) {
			options.BaseEndpoint = aws.String(awsEndpoint)
		})
	} else {
		client = awsSqs.NewFromConfig(cfg)
	}
	return client, nil
}

func getQueueUrl(client *awsSqs.Client, sqsQueueName string, ctx context.Context) (*awsSqs.GetQueueUrlOutput, error) {
	// generate the queue url to publish data to queue via queue name.
	queueURL, err := client.GetQueueUrl(ctx, &awsSqs.GetQueueUrlInput{QueueName: aws.String(os.Getenv(sqsQueueName))})
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQS Queue url, err: %v", err)
	}
	return queueURL, nil
}

func main() {
	endPoint := os.Getenv(awsEndpointURL)
	queueName := os.Getenv(sqsQueueName)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := initClient(ctx, endPoint)
	if err != nil {
		log.Fatalf("error in creating sqs client %s", err)
	}
	queue, err := getQueueUrl(client, queueName, ctx)
	if err != nil {
		log.Fatalf("error getting queue url ,does queue exists?,error - %s", err)
	}
	awsSqsSource := sqs.NewAWSSqsSource(client, queue.QueueUrl)
	err = sourcer.NewServer(awsSqsSource).Start(context.Background())
	if err != nil {
		log.Panic("Failed to start source server : ", err)
	}
}
