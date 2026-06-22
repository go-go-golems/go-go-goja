function apiClient(baseUrl) {
  const fetch = require("fetch");
  return fetch.client()
    .baseUrl(trimRight(baseUrl || "http://127.0.0.1:18792", "/"))
    .acceptJson()
    .expectJson();
}

async function capture(baseUrl, input) {
  return await apiClient(baseUrl)
    .post("/api/capture")
    .json(input)
    .run();
}

async function list(baseUrl) {
  return await apiClient(baseUrl)
    .get("/api/inbox")
    .run();
}

async function archive(baseUrl, id) {
  return await apiClient(baseUrl)
    .post(`/api/inbox/${encodeURIComponent(id)}/archive`)
    .json({})
    .run();
}

function trimRight(value, suffix) {
  while (value.endsWith(suffix)) {
    value = value.slice(0, -suffix.length);
  }
  return value;
}

module.exports = { capture, list, archive };
