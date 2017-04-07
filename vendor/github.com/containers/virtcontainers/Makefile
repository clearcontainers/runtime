all: binaries
	go build ./...
	cd hack/virtc && go build

pause:
	make -C $@

binaries: pause

clean:
	make -C pause clean
	rm -f hack/virtc/virtc

install:
	install -D -m 755 pause/pause /usr/bin/pause

uninstall:
	rm -f /usr/bin/pause

.PHONY: \
	binaries \
	clean \
	install \
	pause \
	uninstall
