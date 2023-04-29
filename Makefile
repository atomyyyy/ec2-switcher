clean:
	@rm -rf dist
	@mkdir -p dist
  
gomon:
	reflex -r '\.go' -s -- sh -c 'make build'
 
lint:
	go fmt src/*.go

build: clean
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/main src/*.go

start:
	sudo sam local start-api