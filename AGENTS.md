# Dotenvless development

Scope: this independent repository only.

- Follow the user's specification and confirmed decisions in docs/design.md.
- Use specification-driven development: specify behavior and acceptance IDs before implementing it.
- Use TDD: execute the failing behavioral test before its implementation; record the actual Red/Green evidence in docs/verification.md.
- Follow M0 through M8 in order. Keep each milestone buildable; update README and commit reviewed project changes after relevant checks pass.
- Discuss material design ambiguities before implementing the dependent behavior. Routine implementation details do not need repeated permission.
- Never read or alter a developer's real Vault in tests. Use generated fake secrets and isolated temporary storage.
- Never log values or echo potentially sensitive invalid CLI arguments or dotenv lines. Do not add a plaintext get command.
- No runtime network access, telemetry, daemon, temporary plaintext env files, or persisted environment changes.
- Prefer small concrete modules; introduce interfaces only for a demonstrated platform or test boundary.
- Run Go formatting, go vet ./..., go test ./..., and a Windows build as appropriate. Record unverified interactive and cross-account cases explicitly.
- Git operations belong to this repository; do not stage or change any parent repository. Never commit toolchains, build/cache outputs, vault data, or real secrets.
