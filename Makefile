all: push

IMAGE_NAME=${IMAGE_BASE_URL}/alert-discovery

local: main.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-w' -o alert-discovery .

alert-discovery: main.go
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o alert-discovery .

container: alert-discovery
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

push: container
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

clean:
	rm -f alert-discovery
