# Security Review: Conductor Loop
**Date:** 2026-02-22
**Task ID:** task-20260222-181500-security-review-multi-agent-rlm
**Reviewer:** Gemini CLI Agent

## RLM Decomposition & Iterations

The review was decomposed into three focused sub-problems to ensure depth and coverage:

1.  **CI/CD & Supply Chain:** Analyzed `.github/workflows`, `Makefile`, `Dockerfile`, `install.sh`, and `scripts/` for pipeline integrity, permission scoping, and secret handling.
2.  **Configuration & Runtime:** Reviewed `config.yaml`, `config.docker.yaml`, `go.mod`, and `package.json` for insecure defaults and dependencies.
3.  **Documentation & Examples:** Audited `docs/`, `examples/`, and `README.md` for accidental secret exposure and unsafe instructions.

This structured approach allowed for parallel analysis (via sub-agents and manual verification) of distinct attack surfaces.

---

## Confirmed Findings

### Critical Severity

#### [CRIT-1] `install.sh`: Missing Binary Integrity Verification
*   **File:** `install.sh` (lines 155-172)
*   **Risk:** Supply Chain Compromise / MITM.
*   **Evidence:** The script downloads binaries from `run-agent.jonnyzzz.com` or GitHub Releases and installs them without verifying a cryptographic signature or checksum.
*   **Impact:** An attacker controlling the CDN or performing a MITM attack could serve a malicious binary which would be immediately executed by the user.

### High Severity

#### [HIGH-1] `install.sh`: Forced "Latest" Version
*   **File:** `install.sh` (lines 80-85)
*   **Risk:** Supply Chain / Availability.
*   **Evidence:** The script strips pinned version tags and forces the use of the `latest` release.
*   **Impact:** Users cannot install specific versions for reproducibility or rollback, exposing them to potentially broken or compromised "latest" releases immediately.

#### [HIGH-2] GitHub Actions Unpinned (Mutable Tags)
*   **File:** `.github/workflows/*.yml` (e.g., `build.yml`, `lint.yml`, `test.yml`)
*   **Risk:** Supply Chain Compromise.
*   **Evidence:** Actions are pinned to mutable tags like `@v4`, `@v2` (e.g., `actions/checkout@v4`, `softprops/action-gh-release@v2`).
*   **Impact:** If an action's tag is updated with malicious code (via account compromise of the action owner), the CI pipeline runs that code with the workflow's permissions.

#### [HIGH-3] `lint.yml`: Unpinned Tool Version
*   **File:** `.github/workflows/lint.yml`
*   **Risk:** Supply Chain Compromise.
*   **Evidence:** `golangci-lint-action` is used with `version: latest`.
*   **Impact:** The pipeline runs an arbitrary (latest) binary version, which could be malicious or introduce breaking changes.

#### [HIGH-4] `build.yml`: Excessive Workflow Permissions
*   **File:** `.github/workflows/build.yml`
*   **Risk:** Privilege Escalation / Lateral Movement.
*   **Evidence:** `permissions: contents: write` is granted at the workflow level.
*   **Impact:** Any compromised step in the workflow (e.g., in the build job) inherits write access to the repository, allowing it to push malicious code or modify releases.

#### [HIGH-5] `Dockerfile`: Mutable Base Image
*   **File:** `Dockerfile`
*   **Risk:** Non-reproducible Builds / Supply Chain.
*   **Evidence:** `FROM alpine:latest`.
*   **Impact:** Builds are not reproducible and susceptible to upstream changes or compromises in the `alpine` latest tag.

### Medium Severity

#### [MED-1] Documentation Encourages Inline Secrets
*   **File:** `docs/user/configuration.md`, `examples/configs/README.md`
*   **Risk:** Credential Leakage.
*   **Evidence:** Examples show `token: sk-xxxxx` directly in `config.yaml`.
*   **Impact:** Users might follow this pattern and commit real secrets to version control.

---

## Rejected / Unconfirmed Findings

*   **Empty `config.yaml`:** The `config.yaml` file in the root is 0 bytes. While unusual, investigation confirmed it is likely a placeholder. `config.docker.yaml` exists and uses secure patterns (`/secrets/`).
*   **Exposed Secrets in Docs:** Grep search found "token:" patterns in documentation, but manual review confirmed they are all placeholders (e.g., `sk-ant-actual-token-here`) or clearly marked as examples.

---

## Prioritized Remediation Tasks

1.  **Immediate Fixes (Critical/High):**
    *   **Secure `install.sh`:** Implement SHA-256 checksum verification for all downloads.
    *   **Fix `install.sh` Versioning:** Allow installing specific versions without redirecting to latest.
    *   **Pin Actions:** Update all GitHub workflows to use SHA-1 hashes for actions (e.g., `uses: actions/checkout@<sha>`).
    *   **Scope Permissions:** Move `permissions: contents: write` in `build.yml` to the specific `publish-release-assets` job.
    *   **Pin Docker Base:** Change `Dockerfile` to use a specific tag/digest (e.g., `alpine:3.21`).

2.  **Process Improvements (Medium):**
    *   **Update Docs:** Remove examples of inline tokens in `config.yaml` and emphasize `token_file` or environment variables.
    *   **CI Hardening:** Implement a policy to prevent merging changes that introduce unpinned actions.

