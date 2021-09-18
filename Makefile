APP_NAME=s3sidecar

.PHONY: help

help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

build: ## Build the container
	docker build -t $(APP_NAME) .

build-multi: ## Build the container for multi platform
	docker buildx build --platform linux/arm/v6,linux/amd64 -t coll97/s3sidecar:latest . --push

deploy: ## Build the container
	docker-compose up -d 
