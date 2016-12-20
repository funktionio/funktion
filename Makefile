build: check funktion-operator

VERSION ?= $(shell cat version/VERSION)
REPO = fabric8io/funktion-operator
TAG = latest
GO := GO15VENDOREXPERIMENT=1 go
BUILD_DIR ?= ./out
NAME = funktion-operator

funktion-operator: $(shell find . -type f -name '*.go')
	go build -o funktion github.com/fabric8io/funktion-operator/cmd/operator

funktion-linux-static: $(shell find . -type f -name '*.go')
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build -o ./out/funktion-operator-linux-amd64 \
		-ldflags "-s" -a -installsuffix cgo \
		github.com/fabric8io/funktion-operator/cmd/operator

check: .check_license

.check_license: $(shell find . -type f -name '*.go' ! -path './vendor/*')
	./scripts/check_license.sh
	touch .check_license

image: check funktion-linux-static
	docker build -t $(REPO):$(TAG) .

test:
	CGO_ENABLED=0 $(GO) test github.com/fabric8io/funktion-operator/cmd github.com/fabric8io/funktion-operator/pkg/funktion

e2e:
	go test -v ./test/e2e/ --kubeconfig "$(HOME)/.kube/config" --operator-image=fabric8io/funktion-operator

clean:
	rm -rf funktion-operator funktion-linux-static .check_license release $(BUILD_DIR)

clean-e2e:
	kubectl delete namespace funktion-operator-e2e-tests

bootstrap:
	$(GO) get -u github.com/Masterminds/glide
	GO15VENDOREXPERIMENT=1 glide install --strip-vendor --strip-vcs --update-vendored
    
.PHONY: build check container e2e clean-e2e clean

out/$(NAME): out/$(NAME)-$(GOOS)-$(GOARCH)
	cp $(BUILD_DIR)/$(NAME)-$(GOOS)-$(GOARCH) $(BUILD_DIR)/$(NAME)

out/$(NAME)-darwin-amd64: gopath $(shell $(GOFILES)) version/VERSION
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build $(BUILDFLAGS) -o $(BUILD_DIR)/$(NAME)-darwin-amd64 github.com/fabric8io/funktion-operator/cmd/operator

out/$(NAME)-linux-amd64: gopath $(shell $(GOFILES)) version/VERSION
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build $(BUILDFLAGS) -o $(BUILD_DIR)/$(NAME)-linux-amd64 github.com/fabric8io/funktion-operator/cmd/operator

out/$(NAME)-windows-amd64.exe: gopath $(shell $(GOFILES)) version/VERSION
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build $(BUILDFLAGS) -o $(BUILD_DIR)/$(NAME)-windows-amd64.exe github.com/fabric8io/funktion-operator/cmd/operator

out/$(NAME)-linux-arm: gopath $(shell $(GOFILES)) version/VERSION
	CGO_ENABLED=0 GOARCH=arm GOOS=linux go build $(BUILDFLAGS) -o $(BUILD_DIR)/$(NAME)-linux-arm github.com/fabric8io/funktion-operator/cmd/operator

.PHONY: release
release: clean bootstrap test cross
	mkdir -p release
	cp out/$(NAME)-*-amd64* release
	cp out/$(NAME)-*-arm* release
	gh-release checksums sha256
	gh-release create fabric8io/$(NAME) $(VERSION) master v$(VERSION)

.PHONY: cross
cross: out/$(NAME)-linux-amd64 out/$(NAME)-darwin-amd64 out/$(NAME)-windows-amd64.exe out/$(NAME)-linux-arm

.PHONY: gopath
gopath: $(GOPATH)/src/$(ORG)