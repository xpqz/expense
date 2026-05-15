BIN := expense

.PHONY: build run fmt vet test tidy clean

build:
	go build -o $(BIN) .

run: build
	./$(BIN) $(ARGS)

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -f $(BIN)
