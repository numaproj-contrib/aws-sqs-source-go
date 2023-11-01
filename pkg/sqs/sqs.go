package sqs

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	sourcesdk "github.com/numaproj/numaflow-go/pkg/sourcer"
	"log"
	"strconv"
	"sync"
	"time"
)

const (
	ApproximateNumberOfMessages           = "ApproximateNumberOfMessages"
	ApproximateNumberOfMessagesNotVisible = "ApproximateNumberOfMessagesNotVisible"
	MaxNumberOfMessages                   = 10 // MaxNumberOfMessages is messages to return From SQS Valid values: 1 to 10. Default: 1
	WaitTimeSeconds                       = 20 // WaitTimeSeconds specifies the time (in seconds) the call waits for a message to arrive in the queue before returning. If a message arrives, the call returns early; if not and the time expires, it returns an empty message list.

)

// AWSSqsSource represents an AWS SQS source with necessary attributes.
type AWSSqsSource struct {
	lock             *sync.Mutex
	queueURL         *string
	toAckSet         map[string]struct{}
	sqsServiceClient sqsiface.SQSAPI
}

// NewAWSSqsSource creates a new AWSSqsSource using the given AWS configuration.
func NewAWSSqsSource(sqsServiceClient sqsiface.SQSAPI, queueName string) (*AWSSqsSource, error) {
	url, err := sqsServiceClient.GetQueueUrl(&sqs.GetQueueUrlInput{QueueName: aws.String(queueName)})
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error in Getting Queue URL %s", err))
	}
	return &AWSSqsSource{
		lock:             new(sync.Mutex),
		queueURL:         url.QueueUrl,
		sqsServiceClient: sqsServiceClient,
		toAckSet:         make(map[string]struct{}),
	}, nil
}

func (s *AWSSqsSource) GetApproximateMessageCount(queryType string) int64 {
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

// Pending  Returns the approximate number of messages available for retrieval from the queue.
func (s *AWSSqsSource) Pending(_ context.Context) int64 {
	return s.GetApproximateMessageCount(ApproximateNumberOfMessages)
}

// Read fetches messages from the SQS queue and sends them to the provided message channel.
func (s *AWSSqsSource) Read(_ context.Context, readRequest sourcesdk.ReadRequest, messageCh chan<- sourcesdk.Message) {
	var readRequestCount int64
	ctx, cancel := context.WithTimeout(context.Background(), readRequest.TimeOut())
	defer cancel()
	// If we have un-acked data, we return without reading any new data.
	if s.GetApproximateMessageCount(ApproximateNumberOfMessagesNotVisible) > 0 {
		return
	}
	/*
		msgResult holds the outcome of the ReceiveMessage API call. This result is of type *sqs.ReceiveMessageOutput
		and encapsulates various fields, among which is the Messages slice. This slice contains the messages
		that have been fetched from the SQS queue.
	*/
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
			s.lock.Lock()
			messageCh <- sourcesdk.NewMessage(
				[]byte(*msgs[i].Body),
				// The ReceiptHandle is a unique identifier for the received message and is required to delete it from the queue
				// partitionId 0 As sqs doesn't have partitions
				sourcesdk.NewOffset([]byte(*msgs[i].ReceiptHandle), "0"),
				time.Now(),
			)
			s.lock.Unlock()
		}
	}
}

// Ack acknowledges a message.
func (s *AWSSqsSource) Ack(_ context.Context, request sourcesdk.AckRequest) {
	// We will delete the message from queue once they are read
	for _, offset := range request.Offsets() {
		// Delete the message from the queue to prevent it from being read again.
		_, err := s.sqsServiceClient.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      s.queueURL,
			ReceiptHandle: aws.String(string(offset.Value())), // Offset value contains the Recipient Handle Value
		})
		if err != nil {
			log.Fatalln("Failed to delete message:", err)
		}
	}
}
