.PHONY: all
all: trace

GOPROXY ?= "https://proxy.golang.org,direct"
.PHONY: trace

trace:
	GOPROXY=$(GOPROXY) \
	go build -ldflags "-s -w" \
       	-o build/trace \
	./cmd/

.PHONY: clean
clean:
	rm -fr ./build/*
