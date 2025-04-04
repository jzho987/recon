build: build-dir
	go build . && mv recon ./bin/recon

build-dir:
	mkdir bin

