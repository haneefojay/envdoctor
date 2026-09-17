# Quickstart

Create `envdoctor.schema.json` with the contract for your application, then validate an explicit dotenv file:

```sh
envdoctor check --env-file .env
```

For CI, validate the process environment and emit JSON:

```sh
envdoctor check --environment --json
```

Create a safe starter contract with:

```sh
envdoctor init
```

Generate `.env.example` from contract metadata:

```sh
envdoctor generate example
```

Actual environment values are never copied into generated files, and secret variables remain blank.
