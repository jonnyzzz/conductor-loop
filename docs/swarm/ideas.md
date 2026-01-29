# Agentic Swarm Design

We notice that agents are much better to work on a smaller tasks. 
The longer it works, the more prompts we give to it, the harder it turns out
for the agent to pursue the initial attention and goals. 

In this project we approach it from the different angle, here is the agent swarm manifesto:
- agent should do a selected task and exit
- task can be of any kind, from coding, to managing
- we build the work as a chunk of recursive tasks
- on each level an agent must decide if it can work on the task or it should delegate the smaller task down to the hierarchy
- we have monitoring tool and see the actual delegation tree
- for communication, we include the [message-bus](../message-bus-mcp) as the solution
- we include a dedicated polling service for the message bus, so that we can monitor and process all requests as sub agents
- message bus helps to track the progress, together with the git history and other artifacts
- agents seem to be quite clumsy with Git, and it means we need to offer Git Pro skills, so it can commit only selected files, without touching other files. Hope it can be done in the upper promts
- we designed the basic schema with ../THE_PROMPT_v5.md and ../run-agent.sh
- there should be no direct communication with agents, instead, one can only write to message bus
- we need monitoring tool, so we track all agent runs for all projects
- run different agents types each time to keep up the work
- we bet on the fact that agent can do small work and exit, every time.
- message bus MCP supports CLI tooling, so we can build piling 
- message bus MCP supports usual REST
- monitoring app should be react/web based to see the tree 
  - 1/3 screen show the tree progress
  - left 1/5 of the screen is message bus messages view
  - down below all agents output, colored per agent
  - all done JetBrains Mono
- the run-agent.sh collects the events of agent start/stop to a dedicated log
- agent start-stop has environment variable parameters to track it
  - the project
  - the task
  - the parent agent id
- the current state of the task is persisted by the agent in the STATE.md file in the task folder so each new started agent is starting from there

The system now looks the following:
INPUTS:
- we have generic prompts to start and initiate the work
- each task goes to a specific project/task folder, there we place the initial TASK.md prompt
- TASK prompt includes the recommented improvements interations
FLOW:
- we start agent with that prompt (and sub agent processes) to review and analyze the current task and figure out how this task can be solved. 
- The Agent is given the root prompt of the SWARM


The root idea of the SWARM:
- agents control everything
- each agent run is persisted and tracked
- agents use MESSAGE-BUS to communicate
- each agent must decide if the task is small enough to work on, otherwise it delegates down recursively
- we track parent-child in the runs


There are following components of the system
- the run-agent.sh script to start agents, which is now managed by the system
- the start-task.sh that asks for a task and starts it with the system
- the monitoring tool, which uses the disk layout and message-bus only (web-ui)

Assumptions:
- message bus MCP is added to all agents
- all agents CLI are configured to run out of the box, tokens provided
- it runs on the same machine as a whole
- monitoring web UI is discussed above





