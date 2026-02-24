# Docs Update R3: Deep Re-check of User Docs

You are a documentation validation+update agent (Round 3). The user docs were updated in Round 2.
Your job is to catch anything Round 2 missed, especially from FACTS-runs-conductor.md which contains
facts extracted from all 125 actual run-agent tasks.

## Files to deep-check and fix

1. `/Users/jonnyzzz/Work/conductor-loop/docs/user/cli-reference.md`
2. `/Users/jonnyzzz/Work/conductor-loop/docs/user/configuration.md`
3. `/Users/jonnyzzz/Work/conductor-loop/docs/user/troubleshooting.md`
4. `/Users/jonnyzzz/Work/conductor-loop/docs/user/faq.md`
5. `/Users/jonnyzzz/Work/conductor-loop/docs/user/web-ui.md`

## Facts sources (read ALL first — pay special attention to runs facts)

```bash
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-user-docs.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-runs-conductor.md
cat /Users/jonnyzzz/Work/conductor-loop/docs/facts/FACTS-suggested-tasks.md
```

## Verify live binary surface again (new subcommands may exist)

```bash
cd /Users/jonnyzzz/Work/conductor-loop
./bin/run-agent --help 2>&1
./bin/run-agent job --help 2>&1
./bin/run-agent job batch --help 2>&1
./bin/run-agent server --help 2>&1
./bin/run-agent server job --help 2>&1
./bin/run-agent server job submit --help 2>&1
./bin/run-agent server bus --help 2>&1
./bin/run-agent server bus read --help 2>&1
./bin/run-agent task --help 2>&1
./bin/run-agent task delete --help 2>&1
./bin/run-agent output --help 2>&1
./bin/run-agent monitor --help 2>&1
./bin/conductor job --help 2>&1
./bin/conductor task --help 2>&1
./bin/conductor project --help 2>&1
./bin/conductor monitor --help 2>&1
./bin/conductor workflow --help 2>&1
./bin/conductor goal --help 2>&1
./bin/conductor bus --help 2>&1
```

## Specific checks from FACTS-runs-conductor.md

From the 125 run history, common user-facing issues were:
- SSE CPU hotspot: is there any user guidance for it?
- Missing run-info.yaml: is there a troubleshooting entry?
- Stale running status: is there a troubleshooting entry?
- Monitor process proliferation: is there guidance?

Check troubleshooting.md for these and add entries if missing.
Check faq.md for any FAQ that answers common issues from the runs.

## Rules

- **Facts override docs**
- Be very precise: only change what is factually wrong or clearly missing
- Add to troubleshooting.md: any recurring issues from FACTS-runs-conductor.md that users would hit
- Do not add placeholder guidance — only real steps that actually work

## Output

Overwrite each file in-place with corrections. Write summary to `output.md`.
