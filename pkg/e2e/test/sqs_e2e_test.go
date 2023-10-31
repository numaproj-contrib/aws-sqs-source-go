package test

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/numaproj-contrib/numaflow-utils-go/testing/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"log"
	"os"
	"testing"
	"time"
)

type SqsSourceSuite struct {
	fixtures.E2ESuite
}

// This sends message to sqs queue
func SendMessage(sess *session.Session, queueUrl string, messageBody string) error {
	sqsClient := sqs.New(sess)
	_, err := sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    &queueUrl,
		MessageBody: aws.String(messageBody),
	})
	log.Println(err)

	return err
}

func CreateAWSSession(accessKey, region, secret, endPoint string) *session.Session {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(accessKey, secret, ""),
			Endpoint:    aws.String(endPoint),
		},
		SharedConfigState: session.SharedConfigDisable,
	}))
	return sess
}

func GetQueueURL(sess *session.Session, queue string) (*sqs.GetQueueUrlOutput, error) {
	sqsClient := sqs.New(sess)

	result, err := sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queue,
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *SqsSourceSuite) TestSqsSource() {
	var message = "aws_Sqs"

	// Get values from environment variables
	accessKey := os.Getenv("AWS_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	secret := os.Getenv("AWS_SECRET")
	queue := os.Getenv("AWS_QUEUE")
	endPoint := os.Getenv("AWS_ENDPOINT")
	sess := CreateAWSSession(accessKey, region, secret, endPoint)
	url, err := GetQueueURL(sess, queue)
	assert.Nil(s.T(), err)

	err = SendMessage(sess, *url.QueueUrl, message)
	assert.Nil(s.T(), err)

	w := s.Given().Pipeline("@testdata/sqs_source.yaml").When().CreatePipelineAndWait()
	w.Expect().VertexPodsRunning()
	defer w.DeletePipelineAndWait()
	w.Expect().SinkContains("redis-sink", message, fixtures.WithTimeout(3*time.Minute))
}

func TestSqsSourceSuite(t *testing.T) {
	suite.Run(t, new(SqsSourceSuite))
}
