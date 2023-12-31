default: help

help: # Show help for each of the Makefile recipes.
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "  \033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; done

setup: # Up localstack and configure all resources with policies.
	docker-compose -f deployments/localstack/docker-compose.yml up --build

send-events: # Send events to event-processor; Default: --qtd-events-send 2 --event-send-semaphore 10
	go run cmd/producer/main.go --qtd-events-send 2 --event-send-semaphore 10

clean-all: # Remove all containers and delete volumes.
	docker-compose -f deployments/localstack/docker-compose.yml down -v

localstack-run-it: # Execute in iterable mode localstack.
	docker exec -it localstack-main bash

list-queues: # Return list of create queues
	docker exec localstack-main awslocal sqs list-queues

test: # Run all test
	go test ./... -coverprofile=coverage.out

test-coverage: test # Run all tests and open coverage per file in browser
	go tool cover -html=coverage.out

mocks-generate: # generate all mocks to use in tests
	mockgen -source internal/repository/events.go -destination internal/repository/mocks/events.go -package mocks

	mockgen -source pkg/aws/messagebroker.go -destination pkg/aws/mocks/messagebroker.go -package mocks
	mockgen -source pkg/aws/dynamodb.go -destination pkg/aws/mocks/dynamodb.go -package mocks
