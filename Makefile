BIN := expense
SHEET_BIN := expense-sheet

.PHONY: build build-merge build-sheet run run-sheet fmt vet test tidy clean

build: build-merge build-sheet

build-merge:
	go build -o $(BIN) .

build-sheet:
	go build -o $(SHEET_BIN) ./cmd/expense-sheet

run: build-merge
	./$(BIN) $(ARGS)

run-sheet: build-sheet
	./$(SHEET_BIN) $(ARGS)

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -f $(BIN) $(SHEET_BIN)
