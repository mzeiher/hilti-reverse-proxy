.PHONY: build
build:
	GOOS=windows GOARCH=amd64 go build -o bin/hilti-amd64-windows.exe main.go
	GOOS=linux GOARCH=amd64 go build -o bin/hilti-amd64-linux main.go