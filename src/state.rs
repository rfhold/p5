use std::collections::HashMap;

use pulumi_automation::{
    event::{EngineEvent, EventType, ResOpFailedDetails, ResOutputsDetails, ResourcePreDetails},
    local::{LocalStack, LocalWorkspace},
    stack::StackChangeSummary,
    workspace::{Deployment, OutputMap, StackSettings, StackSummary},
};
use tui_input::Input;

use crate::widgets::ResourceListState;

type Result<T> = std::result::Result<T, String>;

#[derive(Default)]
pub struct AppState {
    pub context_stack: Vec<AppContext>,
    pub command_prompt: Input,
    pub selected_workspace: Option<WorkspaceState>,

    pub toast: Option<(chrono::DateTime<chrono::Utc>, String)>,

    pub workspaces: Loadable<Vec<LocalWorkspace>>,

    /// workspace paths to their outputs
    pub workspace_store: HashMap<String, WorkspaceOutputs>,
}

impl AppState {
    pub fn background_context(&self) -> AppContext {
        if let Some(context) = self.context_stack.last() {
            if let AppContext::CommandPrompt = context {
                if self.context_stack.len() > 1 {
                    return self.context_stack[self.context_stack.len() - 2].clone();
                }
                return AppContext::Default;
            }
            return context.clone();
        }
        AppContext::Default
    }

    pub fn current_context(&self) -> AppContext {
        self.context_stack.last().cloned().unwrap_or_default()
    }

    pub fn workspace(&self) -> &Loadable<LocalWorkspace> {
        if let Some(state) = &self.selected_workspace {
            if let Some(outputs) = self.workspace_store.get(&state.workspace_path) {
                return &outputs.workspace;
            }
        }

        &Loadable::NotLoaded
    }

    pub fn stack_state_mut(&mut self) -> Option<&mut StackOutputs> {
        if let Some(state) = self.selected_workspace.as_mut() {
            if let Some(outputs) = self.workspace_store.get_mut(&state.workspace_path) {
                if let Some(stack_name) = state.selected_stack.as_mut() {
                    if let Some(stack_outputs) = outputs.stack_store.get_mut(&stack_name.stack_name)
                    {
                        return Some(stack_outputs);
                    }
                }
            }
        }

        None
    }

    pub fn stack_state(&self) -> Option<&StackOutputs> {
        if let Some(state) = self.selected_workspace.as_ref() {
            if let Some(outputs) = self.workspace_store.get(&state.workspace_path) {
                if let Some(stack_name) = state.selected_stack.as_ref() {
                    if let Some(stack_outputs) = outputs.stack_store.get(&stack_name.stack_name) {
                        return Some(stack_outputs);
                    }
                }
            }
        }

        None
    }

    pub fn stack(&self) -> &Loadable<LocalStack> {
        if let Some(stack_outputs) = self.stack_state() {
            return &stack_outputs.stack;
        }

        &Loadable::NotLoaded
    }

    pub fn stack_outputs(&self) -> &Loadable<OutputMap> {
        if let Some(stack_outputs) = self.stack_state() {
            return &stack_outputs.outputs;
        }

        &Loadable::NotLoaded
    }

    pub fn stack_config(&self) -> &Loadable<StackSettings> {
        if let Some(stack_outputs) = self.stack_state() {
            return &stack_outputs.config;
        }

        &Loadable::NotLoaded
    }

    pub fn stack_state_data(&self) -> &Loadable<Deployment> {
        if let Some(stack_outputs) = self.stack_state() {
            return &stack_outputs.state;
        }

        &Loadable::NotLoaded
    }

    pub fn stack_resource_state(
        &mut self,
    ) -> Option<(&Loadable<Deployment>, &mut ResourceListState)> {
        if let Some(workspace_state) = self.selected_workspace.as_mut() {
            if let Some(stack_name) = workspace_state.selected_stack.as_mut() {
                if let Some(stack_outputs) = self
                    .workspace_store
                    .get_mut(&workspace_state.workspace_path)
                {
                    if let Some(stack_outputs) =
                        stack_outputs.stack_store.get_mut(&stack_name.stack_name)
                    {
                        return Some((&stack_outputs.state, &mut stack_name.resource_state));
                    }
                }
            }
        }
        None
    }

    pub fn stack_operation_state(
        &mut self,
    ) -> Option<(&mut OperationProgress, &mut ResourceListState)> {
        if let Some(workspace_state) = self.selected_workspace.as_mut() {
            if let Some(stack_name) = workspace_state.selected_stack.as_mut() {
                if let Some(stack_outputs) = self
                    .workspace_store
                    .get_mut(&workspace_state.workspace_path)
                {
                    if let Some(stack_outputs) =
                        stack_outputs.stack_store.get_mut(&stack_name.stack_name)
                    {
                        if let Some(operation_progress) = &mut stack_outputs.operation {
                            return Some((operation_progress, &mut stack_name.resource_state));
                        }
                    }
                }
            }
        }
        None
    }

    pub fn operation_progress(&self) -> Option<&OperationProgress> {
        if let Some(stack_outputs) = self.stack_state() {
            return stack_outputs.operation.as_ref();
        }
        None
    }

    pub fn stacks(&self) -> &Loadable<Vec<StackSummary>> {
        if let Some(state) = &self.selected_workspace {
            if let Some(outputs) = self.workspace_store.get(&state.workspace_path) {
                return &outputs.stacks;
            }
        }

        &Loadable::NotLoaded
    }

    pub fn workspaces(&self) -> &Loadable<Vec<LocalWorkspace>> {
        &self.workspaces
    }

    pub fn stack_context(&self) -> StackContext {
        if let AppContext::Stack(stack_context) = self.background_context() {
            return stack_context.clone();
        }
        StackContext::Config
    }
}

#[derive(Debug, Clone, Default)]
pub enum Loadable<T> {
    #[default]
    NotLoaded,
    Loading,
    Loaded(T),
}

impl<T> Loadable<T> {
    pub fn is_loaded(&self) -> bool {
        matches!(self, Loadable::Loaded(_))
    }

    pub fn is_loading(&self) -> bool {
        matches!(self, Loadable::Loading)
    }

    pub fn is_not_loaded(&self) -> bool {
        matches!(self, Loadable::NotLoaded)
    }

    pub fn as_mut_or_default(&mut self, default: T) -> &mut T {
        match self {
            Loadable::Loaded(value) => value,
            Loadable::Loading | Loadable::NotLoaded => {
                *self = Loadable::Loaded(default);
                if let Loadable::Loaded(value) = self {
                    value
                } else {
                    unreachable!()
                }
            }
        }
    }

    pub fn as_ref(&self) -> Loadable<&T> {
        match self {
            Loadable::Loaded(value) => Loadable::Loaded(value),
            Loadable::Loading => Loadable::Loading,
            Loadable::NotLoaded => Loadable::NotLoaded,
        }
    }
}

#[derive(Default)]
pub struct WorkspaceOutputs {
    pub workspace: Loadable<LocalWorkspace>,
    pub stacks: Loadable<Vec<StackSummary>>,
    /// stack names to their outputs
    pub stack_store: HashMap<String, StackOutputs>,
}

#[derive(Default)]
pub struct StackOutputs {
    pub stack: Loadable<LocalStack>,
    pub outputs: Loadable<OutputMap>,
    pub config: Loadable<StackSettings>,
    pub state: Loadable<Deployment>,
    pub operation: Option<OperationProgress>,
}

#[derive(Clone, Debug)]
pub enum ProgramOperation {
    Update,
    Destroy,
    Refresh,
}

#[derive(Default, Debug)]
pub struct WorkspaceState {
    pub workspace_path: String,
    pub selected_stack: Option<StackState>,
}

#[derive(Debug)]
pub struct StackState {
    pub stack_name: String,
    pub resource_state: ResourceListState,
}

#[derive(Clone, Default, Debug)]
pub enum AppContext {
    #[default]
    Default,
    CommandPrompt,
    WorkspaceList,
    StackList,
    Stack(StackContext),
}

#[derive(Clone, Default, Debug)]
pub enum StackContext {
    Outputs,
    #[default]
    Config,
    Resources,
    Operation(OperationContext),
}

#[derive(Clone, Debug)]
pub enum OperationContext {
    Details,
    Summary,
    Events,
}

#[derive(Debug, Clone)]
pub struct OperationProgress {
    pub operation: ProgramOperation,
    pub options: Option<OperationOptions>,
    // loaded before executing for user review
    pub change_summary: Loadable<StackChangeSummary>,
    // loaded during execution
    pub events: Loadable<OperationEvents>,
}

impl OperationProgress {
    pub fn is_preview(&self) -> bool {
        if let Some(options) = &self.options {
            options.preview_only
        } else {
            false
        }
    }

    pub fn is_skip_preview(&self) -> bool {
        if let Some(options) = &self.options {
            options.skip_preview
        } else {
            false
        }
    }
}

#[derive(Clone, Default, Debug)]
pub struct OperationOptions {
    pub preview_only: bool,
    pub skip_preview: bool,
}

#[derive(Debug, Clone, Default)]
pub struct OperationEvents {
    pub events: Vec<EngineEvent>,
    pub states: Vec<ResourceOperationState>,
    pub done: bool,
}

#[derive(Debug, Clone)]
pub enum ResourceOperationState {
    InProgress {
        sequence: i64,
        start_time: chrono::DateTime<chrono::Utc>,
        pre_event: ResourcePreDetails,
    },
    Completed {
        sequence: i64,
        start_time: chrono::DateTime<chrono::Utc>,
        end_time: chrono::DateTime<chrono::Utc>,
        pre_event: ResourcePreDetails,
        out_event: ResOutputsDetails,
    },
    Failed {
        sequence: i64,
        start_time: chrono::DateTime<chrono::Utc>,
        end_time: chrono::DateTime<chrono::Utc>,
        pre_event: ResourcePreDetails,
        failed_event: ResOpFailedDetails,
    },
}

impl OperationEvents {
    fn find_in_progress_state_mut(&mut self, urn: &str) -> Result<&mut ResourceOperationState> {
        if let Some((index, _)) = self.states.iter().enumerate().find(|(_, state)| {
            if let ResourceOperationState::InProgress { pre_event, .. } = state {
                pre_event.metadata.urn == urn
            } else {
                false
            }
        }) {
            Ok(&mut self.states[index])
        } else {
            Err("InProgress state not found for the given URN".to_string())
        }
    }

    pub fn apply_event(&mut self, event: EngineEvent) -> Result<()> {
        self.events.push(event.clone());

        let event_time = event
            .timestamp
            .map_or(Some(chrono::Utc::now()), |t| {
                chrono::DateTime::from_timestamp(t, 0)
            })
            .unwrap_or_default();

        match event.event {
            EventType::ResourcePreEvent { details, .. } => {
                let state = ResourceOperationState::InProgress {
                    sequence: event.sequence.unwrap_or_default(),
                    start_time: event_time,
                    pre_event: details,
                };
                self.states.push(state);
            }
            EventType::ResOutputsEvent { details, .. } => {
                let urn = &details.metadata.urn;
                let state = self.find_in_progress_state_mut(urn)?;

                // Transform the InProgress state into a Completed state
                if let ResourceOperationState::InProgress {
                    sequence,
                    start_time,
                    pre_event,
                } = state.clone()
                {
                    *state = ResourceOperationState::Completed {
                        sequence,
                        start_time,
                        end_time: event_time,
                        pre_event,
                        out_event: details,
                    };
                }
            }
            EventType::ResOpFailedEvent { details, .. } => {
                let urn = &details.metadata.urn;
                let state = self.find_in_progress_state_mut(urn)?;

                // Transform the InProgress state into a Failed state
                if let ResourceOperationState::InProgress {
                    sequence,
                    start_time,
                    pre_event,
                } = state.clone()
                {
                    *state = ResourceOperationState::Failed {
                        sequence,
                        start_time,
                        end_time: event_time,
                        pre_event,
                        failed_event: details,
                    };
                }
            }
            _ => {}
        }

        Ok(())
    }
}
