build: check funktion-operator

REPO = fabric8io/funktion-operator
TAG = latest
GO := GO15VENDOREXPERIMENT=1 go

funktion-operator: $(shell find . -type f -name '*.go')
	go build -o funktion-operator github.com/fabric8io/funktion-operator/cmd/operator

funktion-operator-linux-static: $(shell find . -type f -name '*.go')
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build -o funktion-operator-linux-static \
		-ldflags "-s" -a -installsuffix cgo \
		github.com/fabric8io/funktion-operator/cmd/operator

check: .check_license

.check_license: $(shell find . -type f -name '*.go' ! -path './vendor/*')
	./scripts/check_license.sh
	touch .check_license

image: check funktion-operator-linux-static
	docker build -t $(REPO):$(TAG) .

e2e:
	go test -v ./test/e2e/ --kubeconfig "$(HOME)/.kube/config" --operator-image=fabric8io/funktion-operator

clean:
	rm -f funktion-operator funktion-operator-linux-static .check_license

clean-e2e:
	kubectl delete namespace funktion-operator-e2e-tests

bootstrap:
	$(GO) get -u github.com/Masterminds/glide
	GO15VENDOREXPERIMENT=1 glide update --strip-vendor --strip-vcs --update-vendored
    
.PHONY: build check container e2e clean-e2e clean
