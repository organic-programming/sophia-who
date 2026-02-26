---
# Cartouche v1
title: "Sophia Who?"
author:
  name: "B. ALTER"
  copyright: "© 2026 Benoit Pereira da Silva"
created: 2026-02-12
revised: 2026-02-12
lang: en-US
origin_lang: en-US
translation_of: null
translator: null
access:
  humans: true
  agents: false
status: draft
---
# Sophia Who?

> *"Know thyself."*

Sophia is the holon identity manager. She creates, lists, and pins the
civil status of every holon — UUID, name, clade, lineage, and version.

## Commands

```
who new         — create a new holon identity (interactive)
who show <uuid> — display a holon's identity
who list        — list all known holons (local + cached)
who pin <uuid>  — capture version/commit/arch for a holon's binary
```

## Build

```sh
go build -o who ./cmd/who/
```

```sh
 go install ./cmd/who
```

## Organic Programming

This holon is part of the [Organic Programming](https://github.com/organic-programming/seed)
ecosystem. For context, see:

- [Constitution](https://github.com/organic-programming/seed/blob/master/AGENT.md) — what a holon is
