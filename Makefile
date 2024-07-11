build:
	go build -v -o ./_build/scrapollo ./cmd/crawler/main.go

fmt:
	goimports -w ./ && golines -w ./
