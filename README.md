# AWS SQS Source for Numaflow

The AWS SQS Source is a custom user-defined source for [Numaflow](https://numaflow.numaproj.io/) that enables the integration of Amazon Simple Queue Service (SQS) as a source within your Numaflow pipelines.

- [Quick Start](#Quick-Start)
- [Using AWS SQS Source in Your Numaflow Pipeline](#how-to-use-the-aws-sqs-source-in-your-own-numaflow-pipeline)
- [JSON Configuration](#using-json-format-to-specify-the-aws-sqs-source-configuration)
- [Environment Variables Configuration](#using-environment-variables-to-specify-the-aws-sqs-source-configuration)
- [Debugging AWS SQS Source](#debugging-aws-sqs-source)

## Quick Start
This quick start guide will walk you through setting up an AWS SQS source in a Numaflow pipeline.

### Prerequisites
* [Install Numaflow on your Kubernetes cluster](https://numaflow.numaproj.io/quick-start/)
* AWS CLI configured with access to AWS SQS

### Step-by-step Guide

#### 1. Create an AWS SQS Queue

Using AWS CLI or the AWS Management Console, create a new SQS queue.

#### 2. Deploy a Numaflow Pipeline with AWS SQS Source

Save the following Kubernetes manifest to a file (e.g., `sqs-source-pipeline.yaml`), modifying the AWS region, queue name, and AWS credentials accordingly:

```yaml
apiVersion: numaflow.numaproj.io/v1alpha1
kind: Pipeline
metadata:
  name: sqs-source-pipeline
spec:
  vertices:
    - name: sqs-source
      source:
        udflex:
          container:
            image: your-repo/aws-sqs-source-go:v1.0.0
            env:
              - name: AWS_REGION
                value: "us-east-1"
              - name: SQS_QUEUE_NAME
                value: "your-sqs-queue-name"
              - name: AWS_ACCESS_KEY_ID
                valueFrom:
                  secretKeyRef:
                    name: aws-credentials
                    key: accessKeyId
              - name: AWS_SECRET_ACCESS_KEY
                valueFrom:
                  secretKeyRef:
                    name: aws-credentials
                    key: secretAccessKey
    - name: log-sink
      sink:
        log: {}
  edges:
    - from: sqs-source
      to: log-sink
```

Then apply it to your cluster:
```bash
kubectl apply -f sqs-source-pipeline.yaml
```

#### 3. Verify the Pipeline

Check if the pipeline is running:
```bash
kubectl get pipeline sqs-source-pipeline
```

#### 4. Send Messages to the AWS SQS Queue

Using the AWS CLI, send a message to your SQS queue:
```bash
aws sqs send-message --queue-url <YourQueueUrl> --message-body "Hello from SQS"
```

#### 5. Verify the Log Sink

Check the logs of the `log-sink` vertex to see if it received the message from SQS:
```bash
kubectl logs <log-sink-pod-name>
```

You should see output similar to:
```
Message received: Hello from SQS
```

#### 6. Clean up

To delete the Numaflow pipeline:
```bash
kubectl delete pipeline sqs-source-pipeline
```

To delete the SQS queue:
```bash
aws sqs delete-queue --queue-url <YourQueueUrl>
```

Congratulations! You have successfully set up an AWS SQS source in a Numaflow pipeline.

## How to Use the AWS SQS Source in Your Own Numaflow Pipeline

For a detailed guide on integrating AWS SQS as a source in your own Numaflow pipeline, including creating necessary IAM roles and permissions, see [Using AWS SQS Source in Your Numaflow Pipeline](#using-aws-sqs-source-in-your-own-numaflow-pipeline).

## Using JSON Format to Specify the AWS SQS Source Configuration

While YAML is the default configuration format, the AWS SQS source also supports JSON. See the section [Using JSON format to specify the AWS SQS source configuration](#using-json-format-to-specify-the-aws-sqs-source-configuration) for examples and guidance.

## Using Environment Variables to Specify the AWS SQS Source Configuration

The AWS SQS source can alternatively be configured through environment variables. See [Using Environment Variables to Specify the AWS SQS Source Configuration](#using-environment-variables-to-specify-the-aws-sqs-source-configuration) for details.

## Debugging AWS SQS Source

To enable debugging, set the `DEBUG` environment variable to `true` within the AWS SQS source container. This will output additional log messages for troubleshooting.

## Additional Resources

For more information on N

Numaflow and how to use it to process data in a Kubernetes-native way, visit the [Numaflow Documentation](https://numaflow.numaproj.io/). For AWS SQS specific configuration, refer to the [AWS SQS Documentation](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/welcome.html).