# Task: Implement Gemini Stream JSON Fallback

## Context
When running the Gemini CLI on environments with older versions, the command fails if `--output-format stream-json` is used, as this flag is not supported in earlier releases. This causes the application to crash or behave unexpectedly.

## Goal
Ensure the application can gracefully handle environments with older Gemini CLI versions by falling back to supported output formats.

## Objectives
1.  **Version Detection**: Before invoking the CLI with `--output-format stream-json`, check the installed version of the Gemini CLI.
    -   Identify the minimum version that supports `stream-json`.
    -   If the installed version is older, omit the flag or use a supported alternative (e.g., plain text or standard JSON if available).
2.  **Fallback Mechanism**: Alternatively, implement a try-catch block around the execution. If the command fails with an error indicative of "unknown flag" or similar regarding `stream-json`, retry the command without this flag.
3.  **Logging**: Log a warning when falling back to the older format so the user is aware.

## Verification
1.  **Test Environment**: Use or mock an older version of the Gemini CLI that does not support `stream-json`.
2.  **Execution**: Run the application/script invoking the CLI.
3.  **Result**: Confirm that the application does not crash. It should either detect the version beforehand or retry after failure, successfully executing the command with a fallback output format.
