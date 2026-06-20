.PHONY: test test-e2e fmt clean docs

DSN ?= postgresql://docker:docker@localhost:35432/postgres?sslmode=disable
COMPOSE := docker compose -f e2e/docker-compose.yaml
TFPLUGINDOCS_VERSION ?= v0.25.0

test:
	@echo "===> unit tests"
	go test ./...

test-e2e:
	@tfbin=$$(command -v terraform || command -v tofu) || { echo "need terraform or tofu on PATH"; exit 1; }; \
	case "$$tfbin" in *tofu) tofuenv="TF_ACC_PROVIDER_HOST=registry.opentofu.org TF_ACC_PROVIDER_NAMESPACE=hashicorp" ;; *) tofuenv="" ;; esac; \
	trap '$(COMPOSE) down -v' EXIT; \
	echo "===> starting postgres"; \
	$(COMPOSE) up -d --wait --wait-timeout 60 && \
	echo "===> acceptance tests ($$tfbin)" && \
	env TF_ACC=1 TF_ACC_TERRAFORM_PATH="$$tfbin" SQL_DSN='$(DSN)' $$tofuenv go test -count=1 ./e2e/...

docs:
	@echo "===> generating registry docs"
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@$(TFPLUGINDOCS_VERSION) generate --provider-name sql

fmt:
	@echo "===> gofmt"
	gofmt -w -s .

clean:
	@$(COMPOSE) down -v 2>/dev/null || true
	go clean
