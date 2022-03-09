default: push-all

push-all: (push "postgresql")

push APP: (dockerize APP)
	docker push alexeldeib/{{APP}}:latest

dockerize APP:
	docker build -f images/{{APP}}/Dockerfile images/{{APP}} -t alexeldeib/{{APP}}:latest

build:
	go build -o bin/app app/cmd/main.go
