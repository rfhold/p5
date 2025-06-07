use std::collections::HashMap;

use pulumi_automation::{
    event::EngineEvent,
    local::{LocalStack, LocalWorkspace},
    stack::StackChangeSummary,
    workspace::{Deployment, OutputMap, StackSettings},
};
use tui_input::Input;

#[derive(Default)]
pub struct AppState {
    pub context_stack: Vec<AppContext>,
    pub command_prompt: Input,
    pub selected_workspace: Option<WorkspaceState>,

    pub toast: Option<(chrono::DateTime<chrono::Utc>, String)>,

    /// workspace paths to their outputs
    pub workspaces: HashMap<String, WorkspaceOutputs>,
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
            if let Some(outputs) = self.workspaces.get(&state.workspace_path) {
                return &outputs.workspace;
            }
        }

        &Loadable::NotLoaded
    }

    pub fn stack_state_mut(&mut self) -> Option<&mut StackOutputs> {
        if let Some(state) = self.selected_workspace.as_mut() {
            if let Some(outputs) = self.workspaces.get_mut(&state.workspace_path) {
                if let Some(stack_name) = state.selected_stack.as_mut() {
                    if let Some(stack_outputs) = outputs.stacks.get_mut(&stack_name.stack_name) {
                        return Some(stack_outputs);
                    }
                }
            }
        }

        None
    }

    pub fn stack_state(&self) -> Option<&StackOutputs> {
        if let Some(state) = self.selected_workspace.as_ref() {
            if let Some(outputs) = self.workspaces.get(&state.workspace_path) {
                if let Some(stack_name) = state.selected_stack.as_ref() {
                    if let Some(stack_outputs) = outputs.stacks.get(&stack_name.stack_name) {
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

    pub fn stack_preview(&mut self) -> Loadable<&mut StackChangeSummary> {
        match self.stack_state_mut() {
            Some(stack_outputs) => match &mut stack_outputs.preview.change_summary {
                Loadable::Loaded(summary) => Loadable::Loaded(summary),
                Loadable::Loading => Loadable::Loading,
                Loadable::NotLoaded => Loadable::NotLoaded,
            },
            None => Loadable::NotLoaded,
        }
    }

    pub fn stack_update_preview(&self) -> Loadable<&StackChangeSummary> {
        match self.stack_state() {
            Some(stack_outputs) => match &stack_outputs.update.change_summary {
                Loadable::Loaded(summary) => Loadable::Loaded(summary),
                Loadable::Loading => Loadable::Loading,
                Loadable::NotLoaded => Loadable::NotLoaded,
            },
            None => Loadable::NotLoaded,
        }
    }

    pub fn stack_update_events(&self) -> Loadable<&OperationEvents> {
        match self.stack_state() {
            Some(stack_outputs) => match &stack_outputs.update.events {
                Loadable::Loaded(events) => Loadable::Loaded(events),
                Loadable::Loading => Loadable::Loading,
                Loadable::NotLoaded => Loadable::NotLoaded,
            },
            None => Loadable::NotLoaded,
        }
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
}

#[derive(Default)]
pub struct WorkspaceOutputs {
    pub workspace: Loadable<LocalWorkspace>,
    /// stack names to their outputs
    pub stacks: HashMap<String, StackOutputs>,
}

#[derive(Default)]
pub struct StackOutputs {
    pub stack: Loadable<LocalStack>,
    pub outputs: Loadable<OutputMap>,
    pub config: Loadable<StackSettings>,
    pub state: Loadable<Deployment>,
    pub preview: OperationProgress,
    pub update: OperationProgress,
}

#[derive(Default, Debug)]
pub struct WorkspaceState {
    pub workspace_path: String,
    pub selected_stack: Option<StackState>,
}

#[derive(Debug)]
pub struct StackState {
    pub stack_name: String,
}

#[derive(Clone, Default, Debug)]
pub enum AppContext {
    #[default]
    Default,
    CommandPrompt,
    Stack(StackContext),
}

#[derive(Clone, Default, Debug)]
pub enum StackContext {
    Outputs,
    #[default]
    Config,
    Resources,
    Preview,
    Update,
}

#[derive(Debug, Clone, Default)]
pub struct OperationProgress {
    // loaded before executing for user review
    pub change_summary: Loadable<StackChangeSummary>,
    // loaded during execution
    pub events: Loadable<OperationEvents>,
}

#[derive(Debug, Clone, Default)]
pub struct OperationEvents {
    pub events: Vec<EngineEvent>,
    pub done: bool,
}
