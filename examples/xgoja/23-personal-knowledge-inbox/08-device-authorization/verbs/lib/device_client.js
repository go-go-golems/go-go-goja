function apiClient(baseUrl) {
  const fetch = require("fetch");
  return fetch.client()
    .baseUrl(trimRight(baseUrl || "http://127.0.0.1:18795", "/"))
    .acceptJson()
    .expectJson();
}

async function start(baseUrl, clientName) {
  return await apiClient(baseUrl)
    .post("/auth/device/start")
    .json({ clientName: clientName || "personal-inbox-cli", actions: ["user.self.read"] })
    .run();
}

async function token(baseUrl, deviceCode) {
  return await apiClient(baseUrl)
    .post("/auth/device/token")
    .json({ grant_type: "urn:ietf:params:oauth:grant-type:device_code", device_code: deviceCode })
    .run();
}

async function capture(baseUrl, accessToken, input) {
  const fetch = require("fetch");
  return await apiClient(baseUrl)
    .auth(fetch.auth.bearer().token(accessToken))
    .post("/api/programmatic/capture")
    .json(input)
    .run();
}

function trimRight(value, suffix) {
  while (value.endsWith(suffix)) value = value.slice(0, -suffix.length);
  return value;
}

module.exports = { start, token, capture };
