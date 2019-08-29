.PHONY: clean build test

clean:
	rm -rf ./bin/

build:
	mkdir -p bin/
	go build -o bin/crawler

test:
	go test -race -cover -count 50 ./crawler

tests: test
