package test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/numaproj-contrib/numaflow-utils-go/testing/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type SqsSourceSuite struct {
	fixtures.E2ESuite
}

const (
	AWS_ACCESS_KEY = "key"
	AWS_REGION     = "us-east-1"
	AWS_SECRET     = "secret"
	AWS_QUEUE      = "numaflow-test"
	AWS_ENDPOINT   = "http://127.0.0.1:5000"
	BATCH_SIZE     = 10
)

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

// This sends message to sqs queue
func SendMessage(sqsClient *sqs.SQS, queueUrl string, messageBody string) error {
	for i := 0; i < BATCH_SIZE; i++ {
		_, err := sqsClient.SendMessage(&sqs.SendMessageInput{
			QueueUrl:    &queueUrl,
			MessageBody: aws.String(messageBody),
		})
		if err != nil {
			return err
		}
	}
	return nil
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

func (s *SqsSourceSuite) TestSqsSource() {
	var message = "aws_Sqs"

	// Create Moto resources used for mocking aws APIs.
	deleteCMD := fmt.Sprintf("kubectl delete -k ../../config/apps/moto -n %s --ignore-not-found=true", fixtures.Namespace)
	s.Given().When().Exec("sh", []string{"-c", deleteCMD}, fixtures.OutputRegexp(""))
	createCMD := fmt.Sprintf("kubectl apply -k ../../config/apps/moto -n %s", fixtures.Namespace)
	s.Given().When().Exec("sh", []string{"-c", createCMD}, fixtures.OutputRegexp("service/moto created"))
	labelSelector := fmt.Sprintf("app=%s", "moto")
	s.Given().When().WaitForStatefulSetReady(labelSelector)
	s.T().Log("Moto resources are ready")

	s.T().Log("port forwarding moto service")
	time.Sleep(10 * time.Second)
	stopPortForward := s.StartPortForward("moto-0", 5000)
	defer stopPortForward()

	sess := CreateAWSSession(AWS_ACCESS_KEY, AWS_REGION, AWS_SECRET, AWS_ENDPOINT)
	// Create queue client
	sqsClient := sqs.New(sess)
	url, err := setupQueue(sqsClient, AWS_QUEUE)
	assert.Nil(s.T(), err)

	w := s.Given().Pipeline("@testdata/sqs_source.yaml").When().CreatePipelineAndWait()
	w.Expect().VertexPodsRunning()
	err = SendMessage(sqsClient, *url, message)
	assert.Nil(s.T(), err)
	defer w.DeletePipelineAndWait()
	w.Expect().SinkContains("redis-sink", message, fixtures.WithTimeout(1*time.Minute))
}

func TestSqsSourceSuite(t *testing.T) {
	suite.Run(t, new(SqsSourceSuite))
}
