use std::collections::HashMap;

use pulumi_automation::{
    local::{LocalStack, LocalWorkspace},
    workspace::OutputMap,
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

    pub fn stack_state(&self) -> &Loadable<StackOutputs> {
        if let Some(state) = self.selected_workspace.as_ref() {
            if let Some(outputs) = self.workspaces.get(&state.workspace_path) {
                if let Some(stack_name) = state.selected_stack.as_ref() {
                    if let Some(stack_outputs) = outputs.stacks.get(&stack_name.stack_name) {
                        return stack_outputs;
                    }
                }
            }
        }

        &Loadable::NotLoaded
    }

    pub fn stack(&self) -> &Loadable<LocalStack> {
        if let Loadable::Loaded(stack_outputs) = self.stack_state() {
            return &stack_outputs.stack;
        }

        &Loadable::NotLoaded
    }

    pub fn stack_outputs(&self) -> &Loadable<OutputMap> {
        if let Loadable::Loaded(stack_outputs) = self.stack_state() {
            return &stack_outputs.outputs;
        }

        &Loadable::NotLoaded
    }
}

#[derive(Debug, Clone, Default)]
pub enum Loadable<T> {
    #[default]
    NotLoaded,
    Loading,
    Loaded(T),
}

#[derive(Default)]
pub struct WorkspaceOutputs {
    pub workspace: Loadable<LocalWorkspace>,
    /// stack names to their outputs
    pub stacks: HashMap<String, Loadable<StackOutputs>>,
}

#[derive(Default)]
pub struct StackOutputs {
    pub stack: Loadable<LocalStack>,
    pub outputs: Loadable<OutputMap>,
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
}
