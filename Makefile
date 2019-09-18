.PHONY: test cover

TEST_RESULTS ?= coverage

test:
	go test .

cover:
	mkdir -p $(TEST_RESULTS)
	go test -timeout 1m -coverprofile=$(TEST_RESULTS)/cover.out -covermode=atomic .
	go tool cover -html=$(TEST_RESULTS)/cover.out -o $(TEST_RESULTS)/coverage.html
