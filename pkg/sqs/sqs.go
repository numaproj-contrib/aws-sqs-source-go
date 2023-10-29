package sqs

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	sourcesdk "github.com/numaproj/numaflow-go/pkg/sourcer"
	"log"
	"strconv"
	"sync"
	"time"
)

// AWSSqsSource represents an AWS SQS source with necessary attributes.
/**
We are keeping readIdx and toAckSet just for our internal check
*/
type AWSSqsSource struct {
	lock             *sync.Mutex
	queueURL         *string
	toAckSet         map[string]struct{}
	sqsServiceClient sqsiface.SQSAPI
}

// NewAWSSqsSource creates a new AWSSqsSource using the given AWS configuration.
func NewAWSSqsSource(sqsServiceClient sqsiface.SQSAPI, queueName string) *AWSSqsSource {
	url, err := sqsServiceClient.GetQueueUrl(&sqs.GetQueueUrlInput{QueueName: aws.String(queueName)})
	if err != nil {
		return nil
	}
	return &AWSSqsSource{
		lock:             new(sync.Mutex),
		queueURL:         url.QueueUrl,
		sqsServiceClient: sqsServiceClient,
		toAckSet:         make(map[string]struct{}),
	}
}

// Pending returns the number of pending records for the source.
func (s *AWSSqsSource) Pending(_ context.Context) int64 {
	q := &sqs.GetQueueAttributesInput{
		QueueUrl: s.queueURL,
		AttributeNames: []*string{
			aws.String("ApproximateNumberOfMessages"),
		},
	}
	attributes, err := s.sqsServiceClient.GetQueueAttributes(q)
	if err != nil {
		return 0
	}
	// This returns the count of messages that are yet to be deleted
	// they might have been read by the consumer but if they are not deleted they will be present in the queue
	atoi, err := strconv.Atoi(*attributes.Attributes["ApproximateNumberOfMessages"])
	if err != nil {
		return 0
	}
	return int64(atoi)
}

// Read fetches messages from the SQS queue and sends them to the provided message channel.
func (s *AWSSqsSource) Read(_ context.Context, readRequest sourcesdk.ReadRequest, messageCh chan<- sourcesdk.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), readRequest.TimeOut())
	defer cancel()
	// If we have un-acked data, we return without reading any new data.
	if len(s.toAckSet) > 0 {
		return
	}

	// Receive messages from the SQS queue.

	/*msgResult is the result returned from the ReceiveMessage API call.
	It's of type *sqs.ReceiveMessageOutput, which contains several fields,
	including the Messages slice that holds the messages retrieved from the SQS queue

	*/

	msgResult, err := s.sqsServiceClient.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            s.queueURL,
		WaitTimeSeconds:     aws.Int64(20),
		MaxNumberOfMessages: aws.Int64(int64(readRequest.Count())),
	})
	if err != nil {
		log.Fatalln("Error receiving message:", err)

	}
	for i := 0; uint64(i) < readRequest.Count(); i++ {

		msg := msgResult.Messages
		select {
		case <-ctx.Done():
			return
		default:
			s.lock.Lock()
			messageCh <- sourcesdk.NewMessage(
				[]byte(*msg[i].Body), // the body of the message sent in the queue
				// The ReceiptHandle is a unique identifier for the received message and is required to delete it from the queue
				// As sqs doesn't have partitions
				sourcesdk.NewOffset([]byte(*msg[i].ReceiptHandle), "0"),
				time.Now(),
			)
			// Keeping  track of messages read
			s.toAckSet[*msg[i].ReceiptHandle] = struct{}{}
			s.lock.Unlock()
		}
	}

}

// Ack acknowledges a message.
func (s *AWSSqsSource) Ack(_ context.Context, request sourcesdk.AckRequest) {

	// We will delete the message from queue once they are read
	for _, offset := range request.Offsets() {
		// Process the messages...

		// Delete the message from the queue to prevent it from being read again.
		_, err := s.sqsServiceClient.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      s.queueURL,
			ReceiptHandle: aws.String(string(offset.Value())), // Offset value contains the Recipient Handle Value
		})
		if err != nil {
			log.Fatalln("Failed to delete message:", err)
		}
		// Once the message is deleted remove it from the internal memory cache too
		delete(s.toAckSet, string(offset.Value()))
	}

}
