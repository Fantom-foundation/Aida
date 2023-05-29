.PHONY: all
all: test-vm

.PHONY: test-vm

test-vm:
	GIT_COMMIT=`git rev-list -1 HEAD 2>/dev/null || echo ""` && \
	GIT_DATE=`git log -1 --date=short --pretty=format:%ct 2>/dev/null || echo ""` && \
	GOPROXY=$(GOPROXY) \
	go build \
	    -ldflags "-s -w -X github.com/Fantom-foundation/rc-testing/test/vm/test-vm.gitCommit=$${GIT_COMMIT} -X github.com/Fantom-foundation/test/vm/test-vm.gitDate=$${GIT_DATE}" \
	    -o build/test-vm \
	    ./test/vmtest/cmd

.PHONY: clean
clean:
	rm -fr ./build/*
