build:
	go build .

update-local: build
	sudo cp ./recon /usr/local/bin/recon
