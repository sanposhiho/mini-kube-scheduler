.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build -o ./bin/sched ./sched.go

.PHONY: start
start: build
	./hack/start_simulator.sh

# re-generate openapi file for running api-server
.PHONY: openapi
openapi:
	./hack/openapi.sh

.PHONY: docker_build
docker_build:
	docker build -t minisched .

.PHONY: docker_up
docker_up:
	docker-compose up -d

.PHONY: docker_build_and_up
docker_build_and_up: docker_build docker_up

.PHONY: docker_down
docker_down:
	docker-compose down
