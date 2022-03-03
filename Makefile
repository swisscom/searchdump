IMAGE=tools-docker-local.artifactory.swisscom.com/swisscom/searchdump
TAG=1.0.0

.PHONY: docker-build, docker-push

build:
	mkdir -p build/
	CGO_ENABLED=0 go build ./cmd/searchdump -o build/searchdump

clean:
	rm -rf build/

docker-build:
	docker build . -t "$(IMAGE):$(TAG)"

docker-push:
	docker push "$(IMAGE):$(TAG)"

deploy:
	cf push -f manifest.yml
