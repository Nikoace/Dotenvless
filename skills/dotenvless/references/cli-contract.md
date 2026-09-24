# dvl CLI contract

Use this reference for Dotenvless `0.1.0-dev`. Check the installed `dvl --help` and `dvl --version` before applying it to another version. Examples use Windows PowerShell and assume the executable is on PATH. For a quoted executable path, use PowerShell's call operator: `& "..\Dotenvless\bin\dvl.exe" --help`.

## Commands

| Command | Behavior and constraints |
| --- | --- |
| `dvl --help`, `dvl -h`, `dvl --version` | Work outside Git and do not access the Vault. Help is top-level; do not assume `dvl run --help` is a subcommand help form. |
| `dvl status [DIRECTORY]` | Inspect the current/selected nearest Git project without initializing it or changing cwd. Accept one relative/absolute directory, including a subdirectory. Quote paths with spaces; prefix a directory beginning with `-` with `.\`. |
| `dvl init` | Register the current Git project; repeated execution preserves keys. |
| `dvl set KEY` | Read from a Windows interactive console with input hidden; save or replace one key. Reject positional values and redirected/piped stdin. Ctrl+C cancels. |
| `dvl list` | Print sorted key names only; require initialization. An empty list is valid. |
| `dvl unset KEY` | Delete one key; a missing key is an error. |
| `dvl import FILE` | Import the whole file atomically; any existing-key conflict rejects the whole operation. Keep the source file. |
| `dvl import --overwrite FILE` | Allow replacement of all conflicting keys in that import. Use only for an authorized overwrite. |
| `dvl example` | Create `.env.example` at the Git root with sorted `KEY=` lines, no values. Refuse an existing destination; no overwrite flag. |
| `dvl run -- COMMAND [ARG...]` | Inject all project secrets into a child process, inherit cwd/stdin/stdout/stderr, and return its exit code. Require initialization. |

Use the intended application's working directory for every operation except a directory-qualified `status`. Neither `--project`, `--env`, `--json`, `get`, `export`, nor a per-key injection flag is implemented. A child program may have its own flags after the command separator; distinguish those from dvl flags.

## Interpret status and exit codes

Check the fields as well as the exit code:

| Observation | Interpretation |
| --- | --- |
| `Root` / `ID` | The selected project's canonical physical root and derived identity; verify the intended target before mutation. |
| `Vault: not initialized; run dvl init` | No Vault file; status does not create it. |
| `Vault: this project is not initialized; run dvl init` | Vault exists, but this project has no registration. |
| `Vault: initialized; no secrets` | Registered empty project. |
| `Vault: encrypted; Windows DPAPI decryption verified` | This project's stored values were decrypted internally; only their names are displayed. This does not validate credentials with a service. |
| `Git: tracked` | The environment file is tracked even if ignore rules match. |
| `Git: ignored` | The untracked path matches Git ignore rules; this does not remove the file from disk. |
| `Git: not ignored` / `unknown` | The path is unignored, or Git inspection could not establish its state. Neither is a clean bill of health. |

Dotenvless returns 2 for invalid command syntax and 1 for operation failures. `status` may return 0 with no initialization, no keys, or detected/tracked environment files; it is informational, not a CI security gate. `run` forwards the child's exit code, including 1 or 2, so those numbers alone cannot distinguish dvl errors from child failures. Read the value-free diagnostic context.

Status recursively checks filenames `.env` and `.env.*`, excludes `.env.example` and `.env.*.example`, and never reads their contents. It skips directory links and `.git`, `node_modules`, `.venv`, `venv`, `vendor`, `build`, `dist`, `target`, `.gradle`, `.cache`, `.tools`, `__pycache__`. “None found” applies only to this scope; it is not a credential scan.

## Keys, storage, and import

- Use ASCII names matching `[A-Za-z_][A-Za-z0-9_]*`; dvl normalizes them to uppercase. Values can be empty; nonempty presence is not guaranteed by a listed name. Values must be valid UTF-8 without NUL and at most 32 KiB.
- Use Windows DPAPI Current User and the intended user's `%APPDATA%\dotenvless\vault.dat`, outside the project. Do not read or edit the Vault directly. Its metadata and key names are not secret-value encryption guarantees.
- Derive project identity from the canonical physical Git root path. Subdirectories and directory junctions share identity; worktrees, different clones, moves, and renames do not. A branch switch at the same root preserves identity.
- Import UTF-8 with optional BOM, LF/CRLF, comments, `export`, single/double quotes, and multiline quoted values. The source is limited to 1 MiB. Duplicate normalized keys, malformed syntax, invalid text, or oversized values reject the import.
- Do not expect variable interpolation or command execution: `${NAME}` and command-like text remain literal. If an application previously relied on expansion, flag the difference; do not source/evaluate the file to compensate.
- Let dvl report parse location/category without revealing the source line. For a real file's parse failure, ask the user to correct it locally or provide a synthetic reproducer with invented values, never the original line.

## Windows execution

- Resolve commands using the parent process PATH/PATHEXT. A PATH value stored in the Vault does not select a different executable for this launch.
- Start ordinary executables directly. For `.cmd`/`.bat`, dvl uses the system `cmd.exe`. `./gradlew` prefers an adjacent `gradlew.bat`.
- Expect batch paths/arguments containing double quotes, `%`, `!`, `^`, `&`, `|`, `<`, `>`, or control characters to be rejected; batch arguments also reject a trailing backslash. Spaces, Unicode, parentheses, and empty arguments are supported within that contract.
- Do not wrap rejected input in a shell just to bypass validation. If a real task needs shell syntax, establish its non-secret literal command separately and apply normal authorization and output review. Do not concatenate user input into a shell string.
- Merge environment names case-insensitively; project values override inherited duplicates. The parent and persistent environment remain unchanged. The child and descendants can read, log, or transmit the values; dvl does not filter output or restrict their network access.

## Recovery decisions

| Situation | Next action |
| --- | --- |
| Non-Windows/cloud execution or missing local access | Prepare commands for the user's Windows session; mark execution pending. Never upload the Vault. |
| Missing executable | Use the trusted repository's build instructions. Do not invent an installer or assume this skill installs dvl. |
| Not a Git project / wrong root | Select the intended existing Git project. Do not create a Git repository merely to silence the error. |
| Uninitialized project | Run `init` only when setup/migration/run preparation is authorized; report it for inspection-only requests. |
| Missing key | Give the user `dvl set KEY` for hidden local input; then recheck names. |
| No interactive terminal | Hand `set` to the user; do not pipe values, use tool arguments, or write a temporary dotenv file. |
| Import conflict | Preserve existing data; ask about overwrite only if the user has not already authorized that scope. |
| Existing `.env.example` | Preserve it; provide names for a manual merge. Do not force replacement or read unverified values. |
| DPAPI/corruption/APPDATA error | Stop mutations, report the category, and verify Windows identity/project context. Do not reset, relocate, or bypass the Vault. |
| New worktree/moved project | Explain distinct identity; use authorized local re-entry/import, not a raw Vault edit/copy. |
| Rejected batch argument | Identify the unsupported category without echoing sensitive arguments; choose a supported executable/argument form. |
| Child fails or remains running | Report its exit code or readiness state accurately; avoid environment-dump diagnostics. |
