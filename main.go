package main

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
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awsSqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/numaproj/numaflow-go/pkg/sourcer"

	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/sqs"
)

func initSess() *session.Session {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region:      aws.String(os.Getenv("AWS_REGION")),
			Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY"), os.Getenv("AWS_SECRET"), ""),
			Endpoint:    aws.String(os.Getenv("AWS_END_POINT")),
		},
		SharedConfigState: session.SharedConfigDisable,
	}))
	return sess
}

func main() {
	sqsClient := awsSqs.New(initSess())
	awsSqsSource, err := sqs.NewAWSSqsSource(sqsClient, os.Getenv("AWS_QUEUE"))
	if err != nil {
		log.Panic("Failed to Create SQS Source : ", err)
	}
	err = sourcer.NewServer(awsSqsSource).Start(context.Background())
	if err != nil {
		log.Panic("Failed to start source server : ", err)
	}
}
