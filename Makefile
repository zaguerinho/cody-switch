.PHONY: test lint

test: lint
	bats test/

lint:
	shellcheck --severity=error cody-switch
