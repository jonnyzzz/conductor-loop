# Message Bus Object Model - Questions

1. The current Go parser only supports `parents` as a string list. Should we implement object-form parents with `kind`/`meta` and preserve them on read/write?
Answer: yes

2. Should `ISSUE` messages get a dedicated `issue_id` header (alias), or should `msg_id` always serve as the issue identifier?
Answer: issue_id is just an alias for msg_id. Use the first to make it more easy to parse, where necessary. Keep as alias

3. Should dependency kinds such as `depends_on`, `blocks`, `blocked_by`, and `child_of` be enforced or validated by tooling/UI, or remain advisory only?
Answer: only advisory.
