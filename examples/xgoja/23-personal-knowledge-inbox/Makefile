.DEFAULT_GOAL := smoke

.PHONY: smoke step-01 step-02 step-03 step-04 clean

smoke: step-01 step-02 step-03 step-04

step-01:
	$(MAKE) -C 01-minimal-jsverb smoke

step-02:
	$(MAKE) -C 02-hello-web-server smoke

step-03:
	$(MAKE) -C 03-sqlite-cli-inbox smoke

step-04:
	$(MAKE) -C 04-api-client-server smoke

clean:
	$(MAKE) -C 01-minimal-jsverb clean
	$(MAKE) -C 02-hello-web-server clean
	$(MAKE) -C 03-sqlite-cli-inbox clean
	$(MAKE) -C 04-api-client-server clean
