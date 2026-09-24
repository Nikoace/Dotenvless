---
name: dotenvless
description: Use the Dotenvless dvl CLI to manage and inject local Git-project secrets on Windows without reading or displaying their values. Use when asked to configure Dotenvless, migrate an existing .env, inspect secret names or Vault status, create a key-only .env.example, run an app with stored secrets, or troubleshoot missing environment variables in a Dotenvless project. 中文触发：使用 Dotenvless、迁移 .env、检查密钥配置、注入环境变量启动项目。
---

# Dotenvless

Use `dvl` as the interface to secrets. Keep values out of model context, tool arguments, generated code, files, logs, and replies. Follow existing user authorization; do not ask again for routine actions already within the requested scope.

## Establish the execution context

1. Confirm execution is on the user's Windows 10/11 machine under the intended Windows identity, with Git and a trusted `dvl.exe` available. A remote/cloud checkout or Linux/macOS shell is not access to that machine's DPAPI Vault. If unavailable, prepare local PowerShell commands and state that they have not run; do not copy the Vault or secrets to the agent's environment.
2. Run `dvl --version` and `dvl --help`. This skill targets `0.1.0-dev`; if the installed help differs, follow that help and verify the relevant contract before proceeding. Do not invent flags or plaintext retrieval commands. If the executable is missing, use the user's Dotenvless checkout/build instructions, not an unverified download or package name.
3. Identify the intended application directory from the request or workspace. Inspect it with `dvl status "<project-directory>"`, replacing the placeholder with the actual path. Check the returned `Root`, project identity, Vault state, key names, and file/Git notices.
4. Set the execution tool's working directory to the application's required directory within that Git project for every subsequent command. Only `status` accepts a directory argument; it does not change the working directory. Do not initialize the Dotenvless source repository by mistake.

Read [references/cli-contract.md](references/cli-contract.md) for exact syntax, status meanings, import rules, Windows argument restrictions, and recovery paths. The reference is bundled with this skill and requires no source checkout.

## Choose the workflow

### Inspect configuration or missing variables

- Use `dvl status` for the project's state and `dvl list` for sorted key names after initialization.
- Determine required variable names from explicitly identified application configuration/schema files or documentation. Avoid broad content searches over the workspace, credential files, environment dumps, and logs. Do not assume a file is value-free just because it is named `.env.example`.
- Compare names only. A listed key can have an empty value; it does not prove a credential is valid or an application can authenticate. `status` can exit 0 for an uninitialized project or tracked `.env` files.
- Do not run `init`, `import`, `unset`, or the application for a request that only asks for inspection.

### Set up a project or add a key

1. If setup is requested and the project is uninitialized, run `dvl init`; it is idempotent and preserves existing keys.
2. Have the user run `dvl set KEY` in their own Windows interactive terminal and enter the value at the hidden prompt. Give the exact key name and project directory, never ask them to paste the value into chat. Only let the agent initiate the prompt if its terminal explicitly supports private user input that bypasses model/tool transcripts.
3. Do not send a real value through a tool call even when terminal echo is disabled. Do not use positional values, piped/redirected stdin, clipboard reads, temporary `.env` files, or placeholder credentials to bypass the prompt. If an existing key would be replaced, ensure replacement is part of the user's request.
4. After the user completes input, verify the key name via `dvl list` or `dvl status`. Until then, report the input step as pending, not completed.

### Migrate an existing dotenv file

1. Resolve the source filename and target project without reading the file's contents. Ask which source to use only if multiple candidates make the request ambiguous.
2. For an authorized migration, initialize the target if needed, then invoke `dvl import ".env"` (or the actual source path) from the intended application directory. Let `dvl` parse the file locally; do not `cat`, `Get-Content`, upload, or copy it into the conversation.
3. On a key conflict, preserve the Vault and stop the import workflow. Use `dvl import --overwrite ".env"` only when overwriting all conflicting keys is explicitly in scope. It is an all-or-nothing operation, not a merge prompt.
4. Verify names and status. Keep the source file; successful import does not delete it or prove the application uses injected values. If cleanup was requested, first verify application behavior through its normal startup/checks, then remove only the authorized source file. Account for dotenv loaders that may still read the retained file; successful startup alone does not prove migration.
5. Report tracked/not-ignored source files without printing their contents or patches. Do not rewrite Git history or rotate credentials as an implicit part of import; raise these as follow-up work if real credentials were committed.

### Run an application with stored secrets

1. Establish the exact user-authorized command and working directory. Inspect the relevant entry point and scripts, including npm lifecycle hooks or build tasks, for environment dumps, credential logging, and unexpected execution. If the command or its output behavior cannot be established, prepare the command for the user to run locally instead of capturing it with live secrets.
2. Remember that `dvl run` injects **all secrets for this project**, inherits the parent's remaining environment, and passes values to descendants. It offers no per-key filter. Do not launch an AI agent or arbitrary diagnostic code inside that environment.
3. Invoke the executable with separate arguments after `--`. Choose only the command appropriate to this application; these are alternatives:

   ```powershell
   dvl run -- python app.py
   dvl run -- npm run dev
   dvl run -- ./gradlew bootRun
   ```

4. Never validate injection using `env`, `printenv`, `cmd /c set`, `Get-ChildItem Env:`, `process.env`, `os.environ` dumps, value echoes, hashes, or encoded/prefix samples of a real secret. Prefer key-name comparison and the application's ordinary value-free readiness signal. Use generated fake values and isolated temporary storage for implementation tests.
5. Treat stdout/stderr as unredacted application output. Do not enable verbose credential diagnostics or claim that `dvl` masks logs. If a value appears unexpectedly, stop further capture where possible, do not repeat it, and report the exposure without the value.
6. Preserve and report the child exit code. For a long-running server, distinguish a confirmed readiness signal from process completion. Do not claim that a zero exit validates credentials or proves the workspace contains no secrets.

### Create examples or remove keys

- For a requested template, run `dvl example`; it creates sorted `KEY=` lines at the Git root. If `.env.example` already exists, preserve it. Do not delete/truncate it to retry, invent `--force`, or assume its existing contents are safe to read. Report the conflict and offer the key-name list for a deliberate merge.
- Use `dvl unset KEY` only for a user-authorized deletion of that key in the confirmed project. Verify the resulting key list. Do not delete other projects' records or edit `vault.dat` directly.

## Handle boundaries and report results

- Preserve the existing Vault on corruption, decryption errors, or an unexpected identity. Do not reset it, change `APPDATA`, copy ciphertext between projects, or weaken storage checks to make the command succeed.
- Explain missing configuration after a move, rename, new clone, or worktree using physical-path project identity. Re-enter or re-import values locally for that project only when authorized; changing branches in the same directory does not create a new identity.
- Treat the skill as workflow guidance. DPAPI does not isolate other programs running as the same Windows user, and `dvl` is not an agent sandbox, content scanner, or child-output redactor.
- Summarize the project, actions actually performed, present/missing key names, Vault/file notices, child exit/readiness status, and any pending user input. Omit values, complete environment dumps, and unnecessary absolute user paths. Mark anything not executed as unverified.
