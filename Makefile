APPS = ./apps
PACKAGES = ./packages
PROTO = ./proto

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

dev:
	@$(MAKE) install generate-grpc

	@docker compose up --watch
.PHONY: dev

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
