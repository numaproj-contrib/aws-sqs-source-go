package sqs

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
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	sourcesdk "github.com/numaproj/numaflow-go/pkg/sourcer"
)

const (
	ApproximateNumberOfMessages           = "ApproximateNumberOfMessages"
	ApproximateNumberOfMessagesNotVisible = "ApproximateNumberOfMessagesNotVisible"
	MaxNumberOfMessages                   = 10 // MaxNumberOfMessages is messages to return From SQS Valid values: 1 to 10. Default: 1
	WaitTimeSeconds                       = 20 // WaitTimeSeconds specifies the time (in seconds) the call waits for a message to arrive in the queue before returning. If a message arrives, the call returns early; if not and the time expires, it returns an empty message list.
)

// AWSSqsSource represents an AWS SQS source with necessary attributes.
type AWSSqsSource struct {
	queueURL         *string
	sqsServiceClient sqsiface.SQSAPI
}

// NewAWSSqsSource creates a new AWSSqsSource using the given AWS configuration.
func NewAWSSqsSource(sqsServiceClient sqsiface.SQSAPI, queueName string) (*AWSSqsSource, error) {
	url, err := sqsServiceClient.GetQueueUrl(&sqs.GetQueueUrlInput{QueueName: aws.String(queueName)})
	if err != nil {
		return nil, fmt.Errorf("error in Getting Queue URL %s", err)
	}
	log.Println("QueueURL-------", *url.QueueUrl)
	return &AWSSqsSource{
		queueURL:         url.QueueUrl,
		sqsServiceClient: sqsServiceClient,
	}, nil
}

// GetApproximateMessageCount retrieves the approximate count of messages from an AWS SQS queue based on the given queryType.
// The queryType can be either "ApproximateNumberOfMessages" or "ApproximateNumberOfMessagesNotVisible".
// This function fetches the respective attribute from the SQS queue, converts it to an integer, and returns the value.
// If an error occurs during fetching the attributes or converting the value, it logs the error and returns -1.
func (s *AWSSqsSource) getApproximateMessageCount(queryType string) int64 {
	q := &sqs.GetQueueAttributesInput{
		QueueUrl: s.queueURL,
		AttributeNames: []*string{
			aws.String(queryType),
		},
	}
	attributes, err := s.sqsServiceClient.GetQueueAttributes(q)
	if err != nil {
		return -1
	}
	atoi, err := strconv.Atoi(*attributes.Attributes[queryType])
	if err != nil {
		log.Printf("Error in Getting the Pending Items %s", err)
		return -1
	}
	return int64(atoi)
}

// Pending returns the approximate number of messages available for retrieval from the queue.
func (s *AWSSqsSource) Pending(_ context.Context) int64 {
	return s.getApproximateMessageCount(ApproximateNumberOfMessages)
}

// Read fetches messages from the SQS queue and sends them to the provided message channel.
func (s *AWSSqsSource) Read(_ context.Context, readRequest sourcesdk.ReadRequest, messageCh chan<- sourcesdk.Message) {
	var readRequestCount int64 = 10

	ctx, cancel := context.WithTimeout(context.Background(), readRequest.TimeOut())
	defer cancel()
	// If we have un-acked data (data which is received but yet to be deleted from the queue
	if s.getApproximateMessageCount(ApproximateNumberOfMessagesNotVisible) > 0 {
		return
	}

	if readRequest.Count() <= MaxNumberOfMessages {
		readRequestCount = int64(readRequest.Count())
	}
	msgResult, err := s.sqsServiceClient.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            s.queueURL,
		WaitTimeSeconds:     aws.Int64(WaitTimeSeconds),
		MaxNumberOfMessages: aws.Int64(readRequestCount),
	})
	if err != nil {
		log.Fatalln("Error receiving message:", err)
	}
	msgs := msgResult.Messages
	for i := 0; i < len(msgs); i++ {
		select {
		case <-ctx.Done():
			return
		default:
			messageCh <- sourcesdk.NewMessage(
				[]byte(*msgs[i].Body),
				// The ReceiptHandle is a unique identifier for the received message and is required to delete it from the queue
				// partitionId 0 As sqs doesn't have partitions
				sourcesdk.NewOffset([]byte(*msgs[i].ReceiptHandle), "0"),
				time.Now(), // TODO: Send the time when the message was sent to queue
			)
		}
	}
}

// Ack acknowledges the messages that have been read from the SQS queue by deleting them from the queue.
func (s *AWSSqsSource) Ack(_ context.Context, request sourcesdk.AckRequest) {
	for _, offset := range request.Offsets() {
		_, err := s.sqsServiceClient.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      s.queueURL,
			ReceiptHandle: aws.String(string(offset.Value())), // Offset value contains the Recipient Handle Value
		})
		if err != nil {
			log.Fatalln("Failed to delete message:", err)
		}
	}
}
