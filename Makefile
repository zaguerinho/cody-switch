.PHONY: test lint test-shell check-dev-deps bootstrap-dev

test: check-dev-deps lint test-shell

test-shell:
	bats test/

lint:
	shellcheck --severity=error cody-switch

check-dev-deps:
	@missing=""; \
	for cmd in shellcheck bats; do \
		command -v $$cmd >/dev/null 2>&1 || missing="$$missing $$cmd"; \
	done; \
	if [ -n "$$missing" ]; then \
		echo "Missing dev dependencies:$$missing"; \
		echo "Install with: make bootstrap-dev"; \
		echo "macOS: brew install shellcheck bats-core"; \
		echo "Debian/Ubuntu: sudo apt-get update && sudo apt-get install -y shellcheck bats"; \
		exit 1; \
	fi

bootstrap-dev:
	@if command -v brew >/dev/null 2>&1; then \
		brew install shellcheck bats-core; \
	elif command -v apt-get >/dev/null 2>&1; then \
		sudo apt-get update; \
		sudo apt-get install -y shellcheck bats; \
	else \
		echo "Install shellcheck and bats with your package manager."; \
		exit 1; \
	fi
