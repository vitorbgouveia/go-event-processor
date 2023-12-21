default: help

help: # Show help for each of the Makefile recipes.
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "  \033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n\n"; done

setup: # Up local stack and configure all resources with policies.
	docker-compose -f deployments/localstack/docker-compose.yml up --build

clean-all: # Remove all containers and delete volumes.
	docker-compose -f deployments/localstack/docker-compose.yml down -v

localstack-run-it: # Execute in iterable mode localstack.
	docker exec -it localstack-main bash

list-queues: # Return list of create queues
	docker exec localstack-main awslocal sqs list-queues
