default: push-all

push-all: (push "postgresql")

push APP: (dockerize APP)
	docker push alexeldeib/{{APP}}:latest

dockerize APP:
	docker build -f images/{{APP}}/Dockerfile images/{{APP}} -t alexeldeib/{{APP}}:latest

wire:
	cd app/internal && wire 

build: wire
	go build -o bin/app app/cmd/main.go
