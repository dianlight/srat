BINARY_NAME=srat

build: clean format swag test
	mkdir dist 
	go build -o dist/${BINARY_NAME} .

format:
	go mod tidy
	docker run --rm -v $(CURDIR):/code ghcr.io/swaggo/swag:latest fmt

clean:
	go clean
	rm -rf dist

test:
	go test

swag:
	docker run --rm -v $(CURDIR):/code ghcr.io/swaggo/swag:latest init

run: 
	go run . -opt testdata/options.json -conf testdata/config.json
