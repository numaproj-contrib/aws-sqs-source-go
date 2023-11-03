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

package test

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

func SendMessage(sqsClient *sqs.SQS, queueUrl string, messageBody string) error {
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
	time.Sleep(10 * time.Second) // waiting for resources to be ready completely

	s.T().Log("port forwarding moto service")
	stopPortForward := s.StartPortForward("moto-0", 5000)
	time.Sleep(10 * time.Second) //waiting for port forward to be ready
	defer stopPortForward()
	sess := CreateAWSSession(AWS_ACCESS_KEY, AWS_REGION, AWS_SECRET, AWS_ENDPOINT)
	// Create queue client
	sqsClient := sqs.New(sess)
	url, err := setupQueue(sqsClient, AWS_QUEUE)
	assert.Nil(s.T(), err)
	stopChan := make(chan struct{})
	w := s.Given().Pipeline("@testdata/sqs_source.yaml").When().CreatePipelineAndWait()
	w.Expect().VertexPodsRunning()
	go func() {
		for {
			err = SendMessage(sqsClient, *url, message)
			if err != nil {
				log.Fatalf("Error in Sending Message %s", err)
			}
			select {
			case <-stopChan:
				log.Println("Exit sending Message To Queue.....")
				return
			default:
				continue
			}
		}
	}()
	assert.Nil(s.T(), err)
	defer w.DeletePipelineAndWait()
	w.Expect().SinkContains("redis-sink", message, fixtures.WithTimeout(2*time.Minute))
	stopChan <- struct{}{}
}

func TestSqsSourceSuite(t *testing.T) {
	suite.Run(t, new(SqsSourceSuite))
}
