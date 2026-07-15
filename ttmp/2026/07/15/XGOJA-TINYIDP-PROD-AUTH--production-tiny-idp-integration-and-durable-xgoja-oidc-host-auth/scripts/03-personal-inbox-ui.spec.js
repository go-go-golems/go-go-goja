import { expect, test } from "@playwright/test";

const baseURL = required("TINYIDP_APP_BASE_URL");
const login = required("TINYIDP_LOGIN");
const password = required("TINYIDP_PASSWORD");

test.use({
  baseURL,
  ignoreHTTPSErrors: true,
  launchOptions: { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || "/usr/bin/chromium-browser" },
});

test("strict tiny-idp login reaches the inbox UI and approves a device code", async ({ page, request }) => {
  const started = await request.post(`${baseURL}/auth/device/start`, {
    data: { clientName: "playwright-ui-smoke", actions: ["user.self.read"] },
  });
  await expect(started).toBeOK();
  const device = await started.json();
  expect(device.user_code).toBeTruthy();

  await page.goto("/");
  await page.getByRole("link", { name: "Log in" }).click();
  await page.getByLabel("Username").fill(login);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Approve" }).click();

  await expect(page.getByText(new RegExp(`Logged in as ${login}`, "i"))).toBeVisible();
  await page.getByLabel("User code").fill(device.user_code);
  await page.getByRole("button", { name: "Approve device" }).click();
  await expect(page.getByText("Device approved. Return to the CLI and poll for tokens.")).toBeVisible();

  const token = await request.post(`${baseURL}/auth/device/token`, {
    data: { grant_type: "urn:ietf:params:oauth:grant-type:device_code", device_code: device.device_code },
  });
  await expect(token).toBeOK();
  const issued = await token.json();
  expect(issued.access_token).toMatch(/^ggat_/);
  expect(issued.refresh_token).toMatch(/^ggrt_/);
});

function required(name) {
  const value = process.env[name];
  if (!value) throw new Error(`${name} is required`);
  return value;
}
