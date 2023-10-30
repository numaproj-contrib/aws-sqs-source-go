package sqs

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/mocks"
	"github.com/numaproj/numaflow-go/pkg/sourcer"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
	"time"
)

var (
	endPoint  = "http://127.0.0.1:5000"
	region    = "us-east-1"
	accessKey = "access-key"
	secretKey = "secret"
)

var resource *dockertest.Resource
var pool *dockertest.Pool
var sqsClient *sqs.SQS
var queueURl string

func initSess() *session.Session {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
			Endpoint:    aws.String(endPoint),
		},
		SharedConfigState: session.SharedConfigDisable,
	}))
	return sess
}

func sendMessages(client *sqs.SQS, queueURL string, numMessages int) error {
	for i := 1; i <= numMessages; i++ {
		sendParams := &sqs.SendMessageInput{
			QueueUrl:    &queueURL,
			MessageBody: aws.String(fmt.Sprintf("Test Message %d", i)),
		}
		_, err := client.SendMessage(sendParams)
		if err != nil {
			fmt.Printf("Failed to send message %d: %s\n", i, err)
			return err
		}
	}
	return nil
}

func setupQueue(client *sqs.SQS, queueName string) (*string, error) {
	params := &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	}

	response, err := client.CreateQueue(params)
	if err != nil {
		fmt.Println("Error creating queue:", err)
		return nil, err
	}
	return response.QueueUrl, nil
}

func TestMain(m *testing.M) {

	// connect to docker
	p, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not connect to docker ;is it running ? %s", err)
	}
	pool = p
	opts := dockertest.RunOptions{
		Repository:   "motoserver/moto",
		Tag:          "latest",
		Env:          []string{"MOTO_PORT=5000"},
		ExposedPorts: []string{"5000"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5000": {
				{HostIP: "127.0.0.1", HostPort: "5000"},
			},
		},
	}
	resource, err = pool.RunWithOptions(&opts)
	if err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not start resource %s", err)
	}

	if err := pool.Retry(func() error {
		var err error
		awsSession := initSess()
		sqsClient = sqs.New(awsSession)
		queue, err := setupQueue(sqsClient, "numaflow")
		if err != nil {
			log.Fatalf("could not Get Queue URL  %s", err)
		}
		queueURl = *queue
		return nil
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not connect to moto sqs %s", err)
	}
	code := m.Run()
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Couln't purge resource %s", err)
	}
	os.Exit(code)

}
func TestAWSSqsSource_Read2Integ(t *testing.T) {

	err := sendMessages(sqsClient, queueURl, 2)
	assert.Nil(t, err)
	awsSqsSource, err := NewAWSSqsSource(sqsClient, "numaflow")
	assert.Nil(t, err)
	messageCh := make(chan sourcer.Message, 20)
	doneCh := make(chan struct{})

	go func() {
		awsSqsSource.Read(context.TODO(), mocks.ReadRequest{
			CountValue: 2,
			Timeout:    time.Second,
		}, messageCh)
		close(doneCh)
	}()
	<-doneCh
	assert.Equal(t, 2, len(messageCh))

	// Try reading 4 more messages
	// Since the previous batch didn't get acked, the data source shouldn't allow us to read more messages
	// We should get 0 messages, meaning the channel only holds the previous 2 messages
	doneCh2 := make(chan struct{})
	go func() {
		awsSqsSource.Read(context.TODO(), mocks.ReadRequest{
			CountValue: 4,
			Timeout:    time.Second,
		}, messageCh)
		close(doneCh2)
	}()
	<-doneCh2
	assert.Equal(t, 2, len(messageCh))

	// Ack the first batch
	msg1 := <-messageCh
	msg2 := <-messageCh

	awsSqsSource.Ack(context.TODO(), mocks.TestAckRequest{
		OffsetsValue: []sourcer.Offset{msg1.Offset(), msg2.Offset()},
	})
	doneCh3 := make(chan struct{})

	// Send 6 more messages
	err = sendMessages(sqsClient, queueURl, 6)
	assert.Nil(t, err)
	go func() {
		awsSqsSource.Read(context.TODO(), mocks.ReadRequest{
			CountValue: 6,
			Timeout:    time.Second,
		}, messageCh)
		close(doneCh3)
	}()
	<-doneCh3
	assert.Equal(t, 6, len(messageCh))

}
