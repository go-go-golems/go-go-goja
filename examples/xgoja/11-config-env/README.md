# Example: Config and Env Support

This example demonstrates how a generated xgoja binary can read configuration from:

1. A `config.yaml` file in the current working directory
2. Environment variables with the `DEMO_` prefix
3. CLI flags (highest precedence)

## Build

```bash
xgoja build -f xgoja.yaml --xgoja-replace /path/to/go-go-goja
```

## Run with config file

The included `config.yaml` sets `fixture.value = from-config-file`:

```bash
./dist/config-env-demo run script.js
```

Output: `hello from-config-file`

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
