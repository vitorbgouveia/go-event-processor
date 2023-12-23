#!/bin/bash

echo "##### Create SQS queues #####"
awslocal \
  sqs create-queue \
  --queue-name dispatch_event_processor_DLQ

awslocal \
  sqs create-queue \
  --queue-name dispatch_event_processor \
  --attributes '{"RedrivePolicy":"{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:000000000000:dispatch_event_processor_DLQ\",\"maxReceiveCount\":\"3\"}"}'

awslocal \
  sqs create-queue \
  --queue-name dispatch_event_processor_retry \
  --attributes '{"RedrivePolicy":"{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:000000000000:dispatch_event_processor_DLQ\",\"maxReceiveCount\":\"3\"}"}'

echo "##### Create table in DynamoDB #####"
awslocal dynamodb create-table \
    --table-name dispatched_events \
    --key-schema AttributeName=event_id,KeyType=HASH \
    --attribute-definitions AttributeName=event_id,AttributeType=S \
    --billing-mode PAY_PER_REQUEST \
    --region us-east-1

echo "##### Create admin-role #####"
awslocal \
  iam create-role \
  --role-name admin-role \
  --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}'

echo "##### Make S3 bucket #####"
awslocal \
  s3 mb s3://lambda-functions \

echo "##### Copy the lambda function to the S3 bucket #####"
awslocal \
  s3 cp go-lambda.zip s3://lambda-functions \

echo "##### Create the lambda event-processor #####"
awslocal \
  lambda create-function \
  --function-name event-processor \
  --role arn:aws:iam::000000000000:role/admin-role \
  --code S3Bucket=lambda-functions,S3Key=go-lambda.zip \
  --handler event-processor \
  --runtime go1.x \
  --description "lambga go to fan-in fan-out events." \
  --timeout 60 \
  --memory-size 128 \

echo "##### Map the queue to the lambda function #####"
awslocal \
  lambda create-event-source-mapping \
  --function-name event-processor \
  --batch-size 1 \
  --event-source-arn "arn:aws:sqs:us-east-1:000000000000:dispatch_event_processor"

echo "##### Map the queue to the lambda function #####"
awslocal \
  lambda create-event-source-mapping \
  --function-name event-processor \
  --batch-size 1 \
  --event-source-arn "arn:aws:sqs:us-east-1:000000000000:dispatch_event_processor_retry"

echo "##### All resources initialized! 🚀 #####"
