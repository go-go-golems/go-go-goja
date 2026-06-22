.DEFAULT_GOAL := smoke

.PHONY: smoke tinyidp-smoke step-01 step-02 step-03 step-04 step-05 step-06 step-07 step-08 clean

smoke: step-01 step-02 step-03 step-04 step-05 step-06 step-07 step-08

tinyidp-smoke:
	$(MAKE) -C 06-browser-login-keycloak tinyidp-smoke

step-01:
	$(MAKE) -C 01-minimal-jsverb smoke

step-02:
	$(MAKE) -C 02-hello-web-server smoke

step-03:
	$(MAKE) -C 03-sqlite-cli-inbox smoke

step-04:
	$(MAKE) -C 04-api-client-server smoke

step-05:
	$(MAKE) -C 05-embedded-retro-ui smoke

step-06:
	$(MAKE) -C 06-browser-login-keycloak smoke

step-07:
	$(MAKE) -C 07-user-scoped-inbox smoke

step-08:
	$(MAKE) -C 08-device-authorization smoke

clean:
	$(MAKE) -C 01-minimal-jsverb clean
	$(MAKE) -C 02-hello-web-server clean
	$(MAKE) -C 03-sqlite-cli-inbox clean
	$(MAKE) -C 04-api-client-server clean
	$(MAKE) -C 05-embedded-retro-ui clean
	$(MAKE) -C 06-browser-login-keycloak clean
	$(MAKE) -C 07-user-scoped-inbox clean
	$(MAKE) -C 08-device-authorization clean
