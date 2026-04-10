DOCKER ?= docker
DOCKER_IMAGE ?= xoste49/echoip
XGOARCH := amd64
XGOOS := linux
XBIN := $(XGOOS)_$(XGOARCH)/echoip

all: checkfmt vet test install

test:
	go test ./...

vet:
	go vet ./...

checkfmt:
	@sh -c "test -z $$(gofmt -l .)" || { echo "one or more files need to be formatted: try make fmt to fix this automatically"; exit 1; }

fmt:
	gofmt -w .

install:
	go install ./...

geoip-download:
	mkdir -p data
	curl -fsSL -m 60 -o data/city.mmdb "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"
	curl -fsSL -m 60 -o data/country.mmdb "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb"
	curl -fsSL -m 60 -o data/asn.mmdb "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-ASN.mmdb"

docker-build:
	$(DOCKER) build -t $(DOCKER_IMAGE) .

docker-login:
	$(DOCKER) login --username "$(DOCKER_USERNAME)" --password "$(DOCKER_PASSWORD)"

docker-test:
	$(eval CONTAINER=$(shell $(DOCKER) run --rm --detach --publish-all $(DOCKER_IMAGE)))
	$(eval DOCKER_PORT=$(shell $(DOCKER) port $(CONTAINER) | cut -d ":" -f 2))
	curl -fsS -m 5 localhost:$(DOCKER_PORT) > /dev/null; $(DOCKER) stop $(CONTAINER)

docker-push: docker-test docker-login
	$(DOCKER) push $(DOCKER_IMAGE)

xinstall:
	env GOOS=$(XGOOS) GOARCH=$(XGOARCH) go install ./...

publish:
ifndef DEST_PATH
	$(error DEST_PATH must be set when publishing)
endif
	rsync -a $(GOPATH)/bin/$(XBIN) $(DEST_PATH)/$(XBIN)
	@sha256sum $(GOPATH)/bin/$(XBIN)

run:
	go run cmd/echoip/main.go -a data/asn.mmdb -c data/city.mmdb -f data/country.mmdb -H x-forwarded-for -r -s -p
