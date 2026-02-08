GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: lint test build tidy ci release tag last-tag

lint:
	$(GOLANGCI_LINT) run

test:
	$(GO) test -v ./...

build:
	$(GO) build ./...

tidy:
	$(GO) mod tidy
	$(GO) mod verify

ci: lint test build

# Show highest semver tag anywhere in the repo history (defaults to v0.0.0).
last-tag:
	@last=$$(git tag --list "v[0-9]*.[0-9]*.[0-9]*" | sort -V | tail -n1); \
	if [ -z "$$last" ]; then last=v0.0.0; fi; \
	echo $$last

# Create and push a semver tag. If VERSION is not provided, bumps the patch
# from the latest semver tag (or starts at v0.0.1).
# Usage: make tag VERSION=1.2.3
tag:
	@last=$$(git tag --list "v[0-9]*.[0-9]*.[0-9]*" | sort -V | tail -n1); \
	if [ -z "$$last" ]; then last=v0.0.0; fi; \
	version=$${VERSION}; \
	if [ -z "$$version" ]; then \
		base=$${last#v}; IFS=. read -r maj min patch <<< "$$base"; \
		version="$$maj.$$min.$$((patch+1))"; \
		echo "VERSION not set, bumping patch: $$last -> v$$version"; \
	fi; \
	new_tag="v$$version"; \
	if git show-ref --tags "$$new_tag" >/dev/null 2>&1; then echo "Tag $$new_tag already exists locally"; exit 1; fi; \
	if git ls-remote --exit-code --tags origin "$$new_tag" >/dev/null 2>&1; then \
		echo "Tag $$new_tag already exists on origin"; exit 1; \
	fi; \
	git tag "$$new_tag"; \
	git push origin "$$new_tag"

# Convenience alias: run tag target to kick off GitHub Actions release.
# Usage: make release [VERSION=1.2.3]
release: tag
