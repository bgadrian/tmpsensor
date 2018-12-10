# Makefile
source := ./main.go

pre:
	go get -t -v ./...
	go test -race ./...

build: pre
	rm -rf ./build/
	mkdir ./build
	mkdir ./build/linux_amd64
	mkdir ./build/linux_arm64
	# https://golang.org/doc/install/source#environment
	env GO111MODULE=on GOOS=linux GOARCH=amd64 go build -ldflags "-linkmode external -extldflags -static" -o ./build/linux_amd64/tempsensor $(source)
	# must have arm-linux-gnueabi-gcc lib installed eg: sudo apt install gcc-arm-linux-gnueabi
	env GO111MODULE=on CC=arm-linux-gnueabi-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "-linkmode external -extldflags -static" -o ./build/linux_arm64/tempsensor $(source)



