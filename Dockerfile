
FROM golang:1.21 as lambda

ARG PORT=8000
ENV PORT=$PORT

WORKDIR /app/lambda-event-processor

RUN ls -la
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o event-processor ./cmd/lambda

# Install zip in container
RUN apt-get update
RUN apt-get install zip -y
# Enter the src directory, install dependencies, and zip the src directory in the container
RUN zip go-lambda.zip event-processor

FROM localstack/localstack
# Copy lambdas.zip into the localstack directory
COPY --from=lambda /app/lambda-event-processor/go-lambda.zip ./go-lambda.zip