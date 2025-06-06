

## Layout
The repository is split into three sections.

### PUI
Home base for generic tui framework components, basic async orchestration, startup and shutdown, ect.

### Pulumi Automation
A rust analogue to [Pulumi Automation API](https://www.pulumi.com/docs/iac/using-pulumi/automation-api/)

### P5
The main application as the workspace root package


## Adding Functionality

### Events
Events are terminal events, usually key presses. They are used to capture user input, navigate, and execute functions. Events can mutate state, trigger Actions, or do nothing.

### Actions
Actions are synchronous operations that are performed on state. Actions can be triggered by Events or Tasks. They should do minimal computation and generally only work with current state and the action itself.

### Tasks
Tasks are the asynchronous option for state changes. Tasks are triggered by Actions and run outside the state, event, and rendering locks. Tasks are given everything they need to perform their request. When a task needs to update state, it dispatches Actions.

### Layout
Layout widgets are meant for navigational conditional rendering, popups, and guards.

### Commands
Commands are a work in progress.

### Context
Context represents what the user currently has focused. Context is stored in the `context_stack` which facilitates backward navigation, exit behavior, and conditional rendering,

### Loading State
Loadable tasks can be represented in the `Loadable` enum. When there is an entity that needs to be loaded after selecting, we follow a flow to ensure we do not lose our place on exception. We first store the selected "id" before dispatching a Task to fetch the resource. When the Task is finished, it sends back an Action to persist the result. In cases where multiple objects need to be fetched under a single resource, a struct should be created to group outputs, and maps can be used to separate resources. The Actions created to persist these outputs should first attempt to patch existing outputs to avoid overriding.