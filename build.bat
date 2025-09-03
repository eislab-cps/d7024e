@echo off
cd labs

set GOARCH=amd64
SET GOOS=linux
set CGO_ENABLED=0
echo Building Go project...
go build -o ./kademlia_linux.exe && echo Docker stuff... && docker build . -t kadlab

cd ..