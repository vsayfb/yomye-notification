.PHONY: build clean

build:
	@echo "Building Go binary for AWS Lambda (AL2023)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -tags lambda.norpc -o bootstrap ./cmd/notification

	@echo "Packaging Lambda..."
	zip lambda.zip bootstrap

	@rm bootstrap

clean:
	@echo "Cleaning build artifacts..."
	rm -f lambda.zip bootstrap