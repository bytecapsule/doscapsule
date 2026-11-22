# Security Policy

## Supported versions

Only the most recent release receives security fixes.

## Reporting a vulnerability

Please **do not open a public issue** for a security problem.

Use GitHub's private vulnerability reporting:
<https://github.com/bytecapsule/doscapsule/security/advisories/new>

Include, as far as you can:

- what the issue is and which component is affected
- steps to reproduce, or a minimal proof of concept
- the version / commit you tested
- any suggested mitigation

You can expect an initial response within a few days. Once a fix is ready it
is released together with an advisory crediting the reporter (unless you ask
to stay anonymous).

## Scope

In scope: the Go launcher (`cmd/doscapsule`), the embedded web UI
(`frontend/`), and the release tooling in this repository.

Out of scope: DOSBox Staging itself (report to
<https://github.com/dosbox-staging/dosbox-staging>), and any game content you
point the launcher at.

## Security model, briefly

DOS Capsule runs entirely on `127.0.0.1` and never sends data anywhere. It
launches DOSBox as a separate process. The manifest (`@launcher.json`) and the
DOSBox config (`@dosbox.conf`) are user-authored files: a malicious games
folder can execute arbitrary DOS code **inside DOSBox** just by being launched,
which is expected behaviour and not a vulnerability in the launcher.
