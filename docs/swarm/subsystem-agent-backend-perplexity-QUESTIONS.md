# Agent Backend: Perplexity - Questions

- Q: Should the Perplexity adapter use streaming responses for liveness/progress updates? 
  Proposed default:  
  A: Yes, if supported; otherwise emit periodic progress logs. Conduct the research for Perplexity APIs.

- Q: How should Perplexity citations be represented in output.md? 
  Proposed default: Append a "References" section after the main response. 
  A: For Perplexity, we just use the stdout, there is no optput.md, the command output should explain to the calling agent where to look for the output.
