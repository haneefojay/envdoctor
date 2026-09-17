# Contract Reference

The canonical contract file is `envdoctor.schema.json` and uses a restricted JSON Schema Draft 2020-12 profile.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "x-envdoctor-version": "1",
  "type": "object",
  "properties": {
    "PORT": {"type": "integer", "minimum": 1, "default": 3000},
    "DATABASE_URL": {"type": "string", "format": "uri", "x-envdoctor-secret": true}
  },
  "required": ["PORT", "DATABASE_URL"]
}
```

Supported types are `string`, `integer`, `number`, and `boolean`. Supported constraints include enum, const, default, description, pattern, string lengths, numeric bounds, multipleOf, email, uri, and `x-envdoctor-secret`.

Defaults are descriptive only and are never injected. Required and secret variables may not define defaults. Unsupported schema keywords are contract errors.
