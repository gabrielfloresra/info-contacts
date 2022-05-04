.PHONY: build clean deploy

build:
	env GOOS=linux CGO_ENABLED=1 go build -o bin/bot bot/main.go

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --verbose
