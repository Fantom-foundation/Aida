.PHONY: all
all: trace

GOPROXY ?= "https://proxy.golang.org,direct"
.PHONY: trace

trace:
	cd carmen/go ; \
	go generate ./... ; \
	cd ../.. ; \
	GOPROXY=$(GOPROXY) \
	GOPRIVATE=github.com/Fantom-foundation/Carmen \
	go build -ldflags "-s -w" \
       	-o build/trace \
	./cmd/

.PHONY: clean
clean:
	cd carmen/go ; \
	rm -f lib/libstate.so ; \
	cd ../cpp ; \
	bazel clean ; \
	cd ../.. ; \
	rm -fr ./build/*
