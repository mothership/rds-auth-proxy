HOME?=$($HOME)
CERT_DIR?=$(HOME)/.config/rds-auth-proxy
CERTIFICATE_PATH?=$(CERT_DIR)/selfsigned_cert.pem
PRIVATE_KEY_PATH?=$(CERT_DIR)/selfsigned_key.pem

AC_USERNAME?=
DEBUG_TARGET?=rds-auth-proxy-macos-amd

DOCKER_REPO?=ghcr.io/mothership/rds-auth-proxy
DOCKER_TAG?=dev

.PHONY: debug
debug:
	AC_USERNAME=$(AC_USERNAME) goreleaser build -f ./build/goreleaser.yml --snapshot --rm-dist --id $(DEBUG_TARGET) 
	mv dist/rds-auth-proxy-macos-amd_darwin_amd64/rds-auth-proxy /usr/local/bin

.PHONY: debug-release 
debug-release:
	AC_USERNAME=$(AC_USERNAME) goreleaser -f ./build/goreleaser.yml --snapshot --rm-dist
	AC_USERNAME=$(AC_USERNAME) gon ./build/notorizing-config.json

.PHONY: release 
release:
	AC_USERNAME=$(AC_USERNAME) goreleaser -f ./build/goreleaser.yml --rm-dist
	AC_USERNAME=$(AC_USERNAME) gon ./build/notorizing-config.json

gen-certs: debug
	mkdir -p $(CERT_DIR) 
	rm -rf $(CERTIFICATE_PATH) $(PRIVATE_KEY_PATH)
	LOG_LEVEL=debug DEBUG=true rds-auth-proxy gen-cert \
		--certificate $(CERTIFICATE_PATH) \
		--key $(PRIVATE_KEY_PATH) 

.PHONY: docker
docker:
	DOCKER_BUILDKIT=1 docker build \
		-t $(DOCKER_REPO):$(DOCKER_TAG) \
		-f ./build/Dockerfile \
		.

.PHONY: test
test:
	go test -coverprofile=coverage.out ./...

.PHONY: test-cover 
test-cover: test
	go tool cover -html=coverage.out

.PHONY: lint
lint:
	golangci-lint run

.PHONY: it-happen 
it-happen:
	docker-compose up --build
