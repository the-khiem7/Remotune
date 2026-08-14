# Build Artifacts

## Windows executable naming

- Every Windows packaging build must write a versioned executable using the form `out/remotune-v<version>.exe` (for example, `out/remotune-v0.3.0.exe`).
- The build version must be explicit and deterministic. Use a release/semantic version supplied by the user when available.
- If a build is requested without a supplied version, read the semantic version from the active build configuration files, increment its least-significant component (the patch component, for example `0.1.1` → `0.1.2`), persist the incremented value to every active configuration file that carries that build version, and use that exact value for the artifact. If the active configuration files disagree or do not contain a valid semantic version, stop and ask the user instead of guessing.
- Never substitute a timestamp or reuse an existing versioned filename.
- Never overwrite, delete, rename, or force-replace an existing executable that might be running. Windows locks running `.exe` files; if the destination is locked, report the error and ask the user to close the application before retrying.
- After a successful build, report the exact version and artifact path.
- Preserve prior versioned artifacts unless the user explicitly requests cleanup.
- Do not create or overwrite the unversioned compatibility path `out/remotune.exe` automatically. If a compatibility copy is explicitly needed, create it only after the versioned artifact succeeds and only when that destination is not locked.

## Commit composition

When creating a commit for an implementation task, include the related code changes and corresponding docpack/baseline documentation updates in the same commit. Do not split one completed implementation and its docpack update into separate commits.

## Build script changes

When changing build scripts, Docker tasks, or release automation, preserve the versioned-artifact convention above rather than introducing a fixed executable output path.
