//go:build test

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

package sqs

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/numaproj-contrib/numaflow-utils-go/testing/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SqsSourceSuite struct {
	fixtures.E2ESuite
}

const (
	AWS_ACCESS_KEY = "access-key"
	AWS_REGION     = "us-east-1"
	AWS_SECRET     = "access-secret"
	AWS_QUEUE      = "numaflow-test"
	AWS_ENDPOINT   = "http://127.0.0.1:5000"
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

func sendMessage(sqsClient *sqs.SQS, queueUrl string, messageBody string) error {
	_, err := sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    &queueUrl,
		MessageBody: aws.String(messageBody),
	})
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

func (suite *SqsSourceSuite) SetupTest() {

	suite.T().Log("e2e Api resources are ready")

	suite.StartPortForward("e2e-api-pod", 8378)

	// Create Redis Resource
	redisDeleteCmd := fmt.Sprintf("kubectl delete -k ../../config/apps/redis -n %s --ignore-not-found=true", fixtures.Namespace)
	suite.Given().When().Exec("sh", []string{"-c", redisDeleteCmd}, fixtures.OutputRegexp(""))
	redisCreateCmd := fmt.Sprintf("kubectl apply -k ../../config/apps/redis -n %s", fixtures.Namespace)
	suite.Given().When().Exec("sh", []string{"-c", redisCreateCmd}, fixtures.OutputRegexp("service/redis created"))

	suite.T().Log("Redis resources are ready")

	// Create Moto resources used for mocking AWS APIs.
	motoDeleteCmd := fmt.Sprintf("kubectl delete -k ../../config/apps/moto -n %s --ignore-not-found=true", fixtures.Namespace)
	suite.Given().When().Exec("sh", []string{"-c", motoDeleteCmd}, fixtures.OutputRegexp(""))
	motoCreateCmd := fmt.Sprintf("kubectl apply -k ../../config/apps/moto -n %s", fixtures.Namespace)
	suite.Given().When().Exec("sh", []string{"-c", motoCreateCmd}, fixtures.OutputRegexp("service/moto created"))
	motoLabelSelector := fmt.Sprintf("app=%s", "moto")
	suite.Given().When().WaitForStatefulSetReady(motoLabelSelector)
	suite.T().Log("Moto resources are ready")
	time.Sleep(10 * time.Second)

	suite.T().Log("port forwarding moto service")
	suite.StartPortForward("moto-0", 5000)

}

func (suite *SqsSourceSuite) TestSqsSource() {
	var testMessage = "aws_Sqs"

	awsSession := CreateAWSSession(AWS_ACCESS_KEY, AWS_REGION, AWS_SECRET, AWS_ENDPOINT)
	// Create SQS client
	sqsClient := sqs.New(awsSession)
	queueURL, err := setupQueue(sqsClient, AWS_QUEUE)
	assert.Nil(suite.T(), err)

	stopChan := make(chan struct{})
	workflow := suite.Given().Pipeline("@testdata/sqs_source.yaml").When().CreatePipelineAndWait()
	workflow.Expect().VertexPodsRunning()

	go func() {
		for {
			sendErr := sendMessage(sqsClient, *queueURL, testMessage)
			if sendErr != nil {
				log.Fatalf("Error in Sending Message: %s", sendErr)
			}
			select {
			case <-stopChan:
				log.Println("Stopped sending messages to queue.")
				return
			default:
				// Continue sending messages at a specific interval, if needed
				time.Sleep(1 * time.Second)
			}
		}
	}()

	assert.Nil(suite.T(), err)
	defer workflow.DeletePipelineAndWait()
	workflow.Expect().SinkContains("redis-sink", testMessage, fixtures.WithTimeout(2*time.Minute))
	stopChan <- struct{}{}
}

func TestSqsSourceSuite(t *testing.T) {
	suite.Run(t, new(SqsSourceSuite))
}
