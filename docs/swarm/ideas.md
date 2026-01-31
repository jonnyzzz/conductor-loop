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


NOTES:
- MESSAGE-BUS files need to be per-task and project to avoid them mixed
- use MESSAGE-BUS as the way to interfact with an agents
- We need to post user comments to message bus
- Agent may quit before whole work is done, this is where we use Ralph like restarts


The implementation plan:
- make run-agent.sh keep project and task.
- make it store state in the ~/run-agent folder
- we need to have easy to read layout there
- run-agent.sh
  - asserts it has JRUN_TASK_ID JRUN_PROJECT_ID JRUN_ID environment vars set
  - tracks parent-child relation between runs (so we create the tree)
  - allows specify agent and allows to specify "i'm lucky" mode

Layout:
  ~/run-agent
    \ project                          the name of the project, keep it short
        - PROJECT-MESSAGE-BUS.md       the project-related knowledge
        - FACT-<date-time>-<name>.md   we prompt agent to write facts-per-file
        - home-folders.md              we keep the infomration of the project source code folders and additional useful folders
        \ task-<date-time>-<name>
            - TASK-MESSAGE-BUS.md
            - TASK FACT FILES-<date-time>.md
            - TASK_STATE.md            the file where the agent maintains the state
            - runs                     to keep all agent runs there
              -- follow run-agent layout --
              \ <runId>-<date-time>
                - parent-run-id        the file to mark parent run id
                - prompt.md
                - output.md
                - agent-type
                - cwd
                - agent process pid and commandline

post-message.sh -- the tool to post message to the message bus
  - includes type, message, task, project

poll-message.sh -- blocks and waits to read for new meesage (file grows)
  - allow --wait 
  - integrates project and task level messages

The root command is `run-task`. This command is started to 
  - read the TASK.md file with the task description (our use console input)
  - asks a coding agent to create the name of the task
  - asks a coding agent to lookup the project (or make it parameter)
  - this script is responsible to restart the root agent
  - we prompt agent to
  - maintains MESSAGE-BUS processing agents: project and task-level. 
    - Starts agent on each message.
    - Notifies sub agents to deal with it.

There is graphic tool that
 - renders that layout (see above) 
 - shows prompt and output files for each agent
 - allows post messages to *-MESSAGE-BUS files



We need to build application, we start from the graphical application,
it has to be done with react and the best app to draw schemas (e.g. d3 or newer)
we prefer JetBrains mono font. 

The application web ui looks as follows:
- on the left (or on the top) there is a tree view
- Tree roots are the tasks that started by a human
- There is an action "Start new Task"
  - it opens new page
  - it relates to the layout above
  - you select project or type the new one
  - you create task id (or pick existing one)
  - we need to double check if that will be the existing task or a new one
  - there is big window to write/edit prompt of the task
  - you store the text in the local storage to make sure it is never lost
  - when OK is pressed (assiming new task scenario)
    - we specify the project folder to work on  
    - we create the project and task folders (see above)
    - we crate TASK.md file with the user input
    - we start the task with run-task.sh and pass there all we have
    - preference to pick the root agent and rotate
    - the essence of run-tash.sh is while true, basically ralph
    - the main prompt is <<""""
        -   you are going to manage the swarm of agents to achieve the goal.
        -   the goal is explained in the TASK file
        -   during the work you must follow the THE_PROMPT_v5.md approach. 
        -   regularly check MESSAGE-BUS.md for updates,
        -   user interaction is only possible via the MESSAGE-BUS.md
        -   create facts in .md files near MESSAGE-BUS
        -   persist your currect state very short in the STATE file, review that file first
        -   run agent process to cleanup and compact the MESSAGE-BUS, update facts
        -   promote common facts to the project level.
        -   read project level facts.
        -   use message-bus.sh to manage it (you must inline the file)
        -   use full paths for all references
        -   use run-agents.sh to start agents under you in the project folders
        <<""""
    - the task puts the information to the user home folder.
  - the system is resilient, and we regularly check the if the root agent is running,
  - and we start it again if needed.
  - we use the UI with the tree of run-ids to observe what is happening
  - we can send the message to the message bus to update agents with information
  - when an agent is selected we see the output from it
  - on the task node in the tree -- we have multiple nodes -- message-bus, facts, output
  - UPDATE: the tree starts from PROJECTS, not just tasks
  - on the project level you can see message-bus, facts, and tasks
  - there is an action to "start again" on the tasks


MAIN: Prompts are done recursively. So any run-agent agent must be able to 
review and decide if it just works or executes deep. 
We need to introduce a limit of 16 agents to the deep.

MAIN2: Let agent's to split the work by subsystems too. Start one to dig a selected
project module/folder is really a good approach, so the work is consolidated and
the context is much less wasted.


IDEA -- we merge all tools into one docker binary, os it providers
-- `run-agent serve` to start web ui for management (later console mode too)
-- `run-agent task` to start a task
-- `run-agent job` to run agentic job
-- `run-agent bus` to deal with message bus
All commands still prefer user home location for the data storage.
The location is heavily recommended to be under local git (and if so it commit/push often) + GitHub Deploy key is recommeded
The ~/run-agent/config.json is used to configure the tool
  - the projects folder location (default under the ~/run-agent/)
  - the deployment ssh key for the repository
  - all other sensible parameters, also per-project
  - the list of supported agents