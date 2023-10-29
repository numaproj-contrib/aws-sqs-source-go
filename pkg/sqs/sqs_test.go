package sqs

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/numaproj-contrib/aws-sqs-source-go/pkg/mocks"
	"github.com/numaproj/numaflow-go/pkg/sourcer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"strconv"
	"testing"
	"time"
)

func SqsMessageObject() *sqs.ReceiveMessageOutput {
	messages := make([]*sqs.Message, 10)
	for i := 0; i < 10; i++ {
		msg := &sqs.Message{
			Attributes: map[string]*string{
				"ApproximateReceiveCount":          aws.String("1"),
				"ApproximateFirstReceiveTimestamp": aws.String("1635545983"),
			},
			Body:          aws.String("Message Body " + strconv.Itoa(i)),
			MD5OfBody:     aws.String("SomeMD5Value"),
			MessageId:     aws.String("MsgID" + strconv.Itoa(i)),
			ReceiptHandle: aws.String("Receipt" + strconv.Itoa(i)),
		}
		messages[i] = msg
	}

	output := &sqs.ReceiveMessageOutput{
		Messages: messages,
	}
	return output

}

func TestAWSSqsSource_Read(t *testing.T) {

	mockSqsSource := new(mocks.MockSQS)
	sqsQueueUrlOutput := sqs.GetQueueUrlOutput{
		QueueUrl: aws.String("testQueueURL"),
	}
	mockSqsSource.On("GetQueueUrl", mock.Anything).Return(&sqsQueueUrlOutput, nil)
	awsSqsSource := NewAWSSqsSource(mockSqsSource, "testQueue")
	assert.NotNil(t, awsSqsSource)

	// Mocking the Receive Method
	sqsMessageObj := SqsMessageObject()
	mockSqsSource.On("ReceiveMessage", mock.Anything).Return(sqsMessageObj, nil)
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
	doneCh2 := make(chan struct{})

	assert.Equal(t, 2, len(messageCh))

	// Try reading 4 more messages
	// Since the previous batch didn't get acked, the data source shouldn't allow us to read more messages
	// We should get 0 messages, meaning the channel only holds the previous 2 messages
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

	mockSqsSource.On("DeleteMessage", mock.Anything).Return(&sqs.DeleteMessageOutput{}, nil)

	awsSqsSource.Ack(context.TODO(), mocks.TestAckRequest{
		OffsetsValue: []sourcer.Offset{msg1.Offset(), msg2.Offset()},
	})

	doneCh3 := make(chan struct{})

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
