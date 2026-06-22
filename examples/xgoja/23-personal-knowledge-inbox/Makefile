.DEFAULT_GOAL := smoke

.PHONY: smoke step-01 step-02 clean

smoke: step-01 step-02

step-01:
	$(MAKE) -C 01-minimal-jsverb smoke

step-02:
	$(MAKE) -C 02-hello-web-server smoke

clean:
	$(MAKE) -C 01-minimal-jsverb clean
	$(MAKE) -C 02-hello-web-server clean
