IMAGE=tools-docker-local.artifactory.swisscom.com/swisscom/searchdump
TAG=1.0.0

build:
	docker build . -t "$(IMAGE):$(TAG)"

docker-push:
	docker push "$(IMAGE):$(TAG)"

push:
	cf push -f manifest.yml
