APPS = ./apps
PACKAGES = ./packages
PROTO = ./proto

TEMPORAL_ADDRESS ?= temporal:7233
WORKER_DEPLOYMENT_NAME ?= mortgage-worker
DEPLOYMENT_VERSION ?= mortgage-worker-v2

copy-proto:
	@for dir in $(shell ls ${APPS}); do \
		cp -rf ${PROTO} ${APPS}/$$dir || true; \
	done
.PHONY: copy-proto

cruft-update:
ifeq (,$(wildcard .cruft.json))
	@echo "Cruft not configured"
else
	@cruft check || cruft update --skip-apply-ask --refresh-private-variables
endif
.PHONY: cruft-update

deploy:
	@$(MAKE) install

	@docker compose up
.PHONY: deploy

deploy-v2:
	@$(MAKE) install

	@docker compose --profile v2 up
.PHONY: deploy-v2

destroy:
	@docker compose down
.PHONY: destroy

generate-db-migrations:
	$(shell if [ -z "${NAME}" ]; then echo "NAME must be set"; exit 1; fi)
	docker compose run --rm api npm run migration:generate -- ./src/migrations/${NAME}
.PHONY: generate-db-migrations

generate-grpc:
	@rm -Rf ${APPS}/*/src/interfaces
	@rm -Rf ${APPS}/*/v1

	@buf ls-files ${PROTO} && buf generate --template ${PROTO}/buf.gen.yaml ${PROTO} || true
.PHONY: generate-grpc

install: install-js-deps

install-js-deps:
	@for dir in $(shell ls ${APPS}/*/package.json ${PACKAGES}/*/package.json); do \
		cd $$(dirname $$dir); \
		echo "Installing $$PWD"; \
		npm ci; \
		cd - > /dev/null; \
	done

	@echo "Installing ${PWD}"
	@npm ci
.PHONY: install-js-deps

# set-worker-version runs the worker-version helper container with
# BOOTSTRAP_ONLY=0, so it actively promotes or rolls back to the requested
# DEPLOYMENT_VERSION. The Temporal server container is not touched.
#
# Examples:
#   make set-worker-version                                          # promote v2 (default)
#   DEPLOYMENT_VERSION=mortgage-worker-v1 make set-worker-version    # roll back to v1
set-worker-version:
	@docker compose run --rm \
		-e BOOTSTRAP_ONLY=0 \
		-e TEMPORAL_ADDRESS=$(TEMPORAL_ADDRESS) \
		-e WORKER_DEPLOYMENT_NAME=$(WORKER_DEPLOYMENT_NAME) \
		-e DEPLOYMENT_VERSION=$(DEPLOYMENT_VERSION) \
		worker-version
.PHONY: set-worker-version
