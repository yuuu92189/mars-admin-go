# ADR 0006: Provider Failures Are Explicit

Status: accepted

Third-party providers never return synthetic success. Missing credentials produce a typed not-configured error. The console SMS provider is enabled only outside production and logs no full phone number or code. Payment, social login and push adapters must be added only against verified provider behavior and contracts.
