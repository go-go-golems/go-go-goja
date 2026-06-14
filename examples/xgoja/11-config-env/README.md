# Example: Config and Env Support

This example demonstrates how a generated xgoja binary can read configuration from:

1. A `config.yaml` file in the current working directory
2. Environment variables with the `DEMO_` prefix
3. CLI flags (highest precedence)

## Build

```bash
xgoja doctor -f xgoja.yaml
xgoja build -f xgoja.yaml
```

The spec uses `workspace.mode: auto`, so local module replacements come from the repository workspace when available.

## Run with config file

The included `config.yaml` sets `fixture.value = from-config-file`:

```bash
./dist/config-env-demo run script.js
```

Output: `hello from-config-file`

You can also load an explicit config file when the `explicit` layer is enabled in `xgoja.yaml`:

```bash
./dist/config-env-demo run --config-file config.yaml script.js
```

The flag name is Glazed's `--config-file` (not `--config`).

## Override with environment variable

```bash
DEMO_FIXTURE_VALUE=from-env ./dist/config-env-demo run script.js
```

Output: `hello from-env`

## Override with CLI flag (highest precedence)

```bash
DEMO_FIXTURE_VALUE=from-env ./dist/config-env-demo run --fixture-value from-flag script.js
```

Output: `hello from-flag`

## Precedence

Config file < Environment variables < CLI flags
