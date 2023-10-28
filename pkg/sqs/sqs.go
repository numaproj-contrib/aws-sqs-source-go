package sqs

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/config"
	sourcesdk "github.com/numaproj/numaflow-go/pkg/sourcer"
)

// AWSSqsSource represents an AWS SQS source with necessary attributes.
type AWSSqsSource struct {
	session          *session.Session
	lock             *sync.Mutex
	queueURL         *string
	sqsServiceClient *sqs.SQS // This is missing in the original code. Ensure it is initialized.
}

// initSQS initializes an AWS session for SQS using the given region, access key, and secret.
func initSess(region string, accessKey string, awsSecret string) *session.Session {
	awsconfig := aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, awsSecret, ""),
	}
	return session.Must(session.NewSession(&awsconfig))
}

// NewAWSSqsSource creates a new AWSSqsSource using the given AWS configuration.
func NewAWSSqsSource(config config.AwsConfig) *AWSSqsSource {
	sess := initSess(config.AwsRegion, config.AwsAccessKey, config.AwsSecret) // Fixed incorrect parameter.
	svc := sqs.New(sess)
	url, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{QueueName: aws.String("numaflow")})
	if err != nil {
		return nil
	}
	if sess != nil {
		return &AWSSqsSource{
			session:          sess,
			lock:             new(sync.Mutex),
			queueURL:         url.QueueUrl,
			sqsServiceClient: svc,
		}
	}
	return nil
}

// Pending returns the number of pending records for the source.
// Currently, it always returns zero.
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
	log.Println(attributes)
	return 0
}

// GetQueueURL retrieves the URL for the given SQS queue.
func GetQueueURL(sess *session.Session, queue *string) (*sqs.GetQueueUrlOutput, error) {
	svc := sqs.New(sess)
	return svc.GetQueueUrl(&sqs.GetQueueUrlInput{

		QueueName: queue,
	})
}

// Read fetches messages from the SQS queue and sends them to the provided message channel.
func (s *AWSSqsSource) Read(_ context.Context, readRequest sourcesdk.ReadRequest, messageCh chan<- sourcesdk.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), readRequest.TimeOut())
	defer cancel()

	//timeOut := int64(5)

	// Receive messages from the SQS queue.
	msgResult, err := s.sqsServiceClient.ReceiveMessage(&sqs.ReceiveMessageInput{

		QueueUrl:        s.queueURL,
		WaitTimeSeconds: aws.Int64(20),
	})
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Println(msgResult)

	// Process the messages.
	for _, msg := range msgResult.Messages {
		select {
		case <-ctx.Done():
			return
		default:
			s.lock.Lock()
			messageCh <- sourcesdk.NewMessage(
				[]byte(msg.String()),
				sourcesdk.NewOffset([]byte(msg.String()), "0"),
				time.Now(),
			)
			s.lock.Unlock()
		}
	}
}

// Ack acknowledges a message.
// Currently, it's a stub and does nothing.
func (s *AWSSqsSource) Ack(_ context.Context, request sourcesdk.AckRequest) {

}
