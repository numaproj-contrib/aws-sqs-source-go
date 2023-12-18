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
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sourcesdk "github.com/numaproj/numaflow-go/pkg/sourcer"
	"log"
	"strconv"
	"sync"
	"time"
)

const (
	approximateNumberOfMessages = "ApproximateNumberOfMessages"
	maxNumberOfMessages         = 10 // maxNumberOfMessages is messages to return From SQS Valid values: 1 to 10. Default: 1
	waitTimeSeconds             = 20 // waitTimeSeconds specifies the time (in seconds) the call waits for a message to arrive in the queue before returning. If a message arrives, the call returns early; if not and the time expires, it returns an empty message list.
)

// AWSSqsSource represents an AWS SQS source with necessary attributes.
type AWSSqsSource struct {
	queueURL         *string
	toAckSet         map[string]struct{}
	sqsServiceClient *sqs.Client
	lock             *sync.Mutex
}

func NewAWSSqsSource(client *sqs.Client, queueURL *string) *AWSSqsSource {
	return &AWSSqsSource{
		sqsServiceClient: client,
		queueURL:         queueURL,
		toAckSet:         make(map[string]struct{}),
		lock:             new(sync.Mutex),
	}
}

// getApproximateMessageCount retrieves the approximate count of messages from an AWS SQS queue based on the given queryType.
// The queryType can be either "approximateNumberOfMessages" or "approximateNumberOfMessagesNotVisible".
// This function fetches the respective attribute from the SQS queue, converts it to an integer, and returns the value.
// If an error occurs during fetching the attributes or converting the value, it logs the error and returns -1.
func (s *AWSSqsSource) getApproximateMessageCount(queryType string) (int64, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q := &sqs.GetQueueAttributesInput{
		QueueUrl: s.queueURL,
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeName(queryType),
		},
	}
	attributes, err := s.sqsServiceClient.GetQueueAttributes(ctx, q)
	if err != nil {
		return -1, err
	}
	atoi, err := strconv.Atoi(attributes.Attributes[queryType])
	if err != nil {
		return -1, err
	}
	return int64(atoi), nil
}

// Pending returns the approximate number of messages available for retrieval from the queue.
func (s *AWSSqsSource) Pending(_ context.Context) int64 {
	count, err := s.getApproximateMessageCount(approximateNumberOfMessages)
	if err != nil {
		log.Fatalf("error getting approximate number of messages from queue %s", err)
		return -1
	}
	return count
}

// Read fetches messages from the SQS queue and sends them to the provided message channel.
func (s *AWSSqsSource) Read(_ context.Context, readRequest sourcesdk.ReadRequest, messageCh chan<- sourcesdk.Message) {
	// If we have un-acked data, we return without reading any new data.
	if len(s.toAckSet) > 0 {
		return
	}
	var readRequestCount int32 = 10
	ctx, cancel := context.WithTimeout(context.Background(), readRequest.TimeOut())
	defer cancel()

	if readRequest.Count() <= maxNumberOfMessages {
		readRequestCount = int32(readRequest.Count())
	}
	msgResult, err := s.sqsServiceClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            s.queueURL,
		WaitTimeSeconds:     waitTimeSeconds,
		MaxNumberOfMessages: readRequestCount,
	})
	if err != nil {
		log.Printf("error receiving message:%s", err)
		return
	}
	msgs := msgResult.Messages
	for i := 0; i < len(msgs); i++ {
		s.lock.Lock()
		messageCh <- sourcesdk.NewMessage(
			[]byte(*msgs[i].Body),
			// The ReceiptHandle is a unique identifier for the received message and is required to delete it from the queue
			// partitionId 0 As sqs doesn't have partitions
			sourcesdk.NewOffset([]byte(*msgs[i].ReceiptHandle), "0"),
			time.Now(),
		)
		s.toAckSet[*msgs[i].ReceiptHandle] = struct{}{}
		s.lock.Unlock()
	}
}

// Ack acknowledges the messages that have been read from the SQS queue by deleting them from the queue.
func (s *AWSSqsSource) Ack(ctx context.Context, request sourcesdk.AckRequest) {
	for _, offset := range request.Offsets() {
		_, err := s.sqsServiceClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      s.queueURL,
			ReceiptHandle: aws.String(string(offset.Value())), // Offset value contains the Recipient Handle Value
		})
		if err != nil {
			log.Fatalln("Failed to delete message:", err)
		}
		delete(s.toAckSet, string(offset.Value()))
	}
}
