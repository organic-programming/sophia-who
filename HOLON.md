---
# Holon Identity v1
uuid: "b00932e5-49d4-4724-ab4b-e2fc9e22e108"
given_name: "Sophia"
family_name: "Who?"
motto: "Know thyself."
composer: "B. ALTER"
clade: "deterministic/pure"
status: draft
born: "2026-02-12"

# Lineage
parents: []
reproduction: "manual"

# Optional
aliases: ["who", "sophia"]

# Metadata
generated_by: "manual"
lang: "go"
proto_status: draft
---

# Sophia Who?

> *"Know thyself."*

## Description

Sophia Who? is the first holon — the primordial identity-maker. She is
a Go CLI that interactively guides a composer (human or agent) through
the creation of a holon identity card (`HOLON.md`).

She has no parents. Every other holon's identity passes through her.

## Commands

```
who new         — create a new holon identity (interactive)
who show <uuid> — display a holon's identity
who list        — list all known holons (local + cached), labeling origin
```

## Contract

- Proto file: `sophia_who.proto`
- Service: `SophiaWhoService`
- RPCs: `CreateIdentity`, `ShowIdentity`, `ListIdentities`
