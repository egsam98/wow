help: ## Show this help.
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0

lint: ## Run linter
	go mod tidy
	golangci-lint run

run: ## Run services from docker-compose.yaml
	docker-compose up --remove-orphans
