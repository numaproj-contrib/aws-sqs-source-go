package e2e

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

	return err
}

func CreateAWSSession(accessKey, region, secret string) *session.Session {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:      aws.String(region),
			Credentials: credentials.NewStaticCredentials(os.Getenv(accessKey), os.Getenv(secret), ""),
			Endpoint:    aws.String(os.Getenv("AWS_END_POINT")),
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
	sess := CreateAWSSession("AKIAS2LNCO4EY3JOZMTY", "us-east-1", "jEhARJ4ltSq7MvjgDw8kgNKZnk4228jCDDLzmEv0")
	url, err := GetQueueURL(sess, "numaflow")
	assert.Nil(s.T(), err)

	err = SendMessage(sess, *url.QueueUrl, message)
	assert.Nil(s.T(), err)

	w := s.Given().Pipeline("@testdata/sqs_source.yaml").When().CreatePipelineAndWait()
	w.Expect().VertexPodsRunning()
	defer w.DeletePipelineAndWait()
	w.Expect().SinkContains("redis-sink", message, fixtures.WithTimeout(2*time.Minute))

}

func TestSqsSourceSuite(t *testing.T) {
	suite.Run(t, new(SqsSourceSuite))
}
