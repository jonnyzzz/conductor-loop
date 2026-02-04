# Agent Backend: Gemini - Questions

- Q: What environment variable name does the Gemini CLI expect for API credentials? 
  A: GEMINI_API_KEY.

- Q: Does the Gemini CLI support streaming/unbuffered stdout so the UI can show live progress? 
  Proposed default: Yes; enforce unbuffered output if supported. 
  A: Look at the ../run-agent.sh and conduct enough experimentation to determine the right answers. Look at the project sources.
