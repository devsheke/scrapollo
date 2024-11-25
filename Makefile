build:
	go build -v -o ./_build/scrapollo ./cmd/cli/main.go

fmt:
	goimports -w ./ && golines -w ./
