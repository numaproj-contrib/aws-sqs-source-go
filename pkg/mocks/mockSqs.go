package mocks

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/stretchr/testify/mock"
)

type MockSQS struct {
	mock.Mock
	sqsiface.SQSAPI
}

func (m *MockSQS) SendMessage(in *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	args := m.Called(in)
	return args.Get(0).(*sqs.SendMessageOutput), args.Error(1)
}

func (m *MockSQS) GetQueueUrl(in *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	args := m.Called(in)
	return args.Get(0).(*sqs.GetQueueUrlOutput), args.Error(1)
}

func (m *MockSQS) ReceiveMessage(in *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(in)
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)

}

func (m *MockSQS) DeleteMessage(in *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	args := m.Called(in)
	return args.Get(0).(*sqs.DeleteMessageOutput), args.Error(1)
}

func (m *MockSQS) GetQueueAttributes(in *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	args := m.Called(in)
	return args.Get(0).(*sqs.GetQueueAttributesOutput), args.Error(1)

}
