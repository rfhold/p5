use std::sync::Arc;

use crossterm::event::{Event, KeyCode};
use operation::OperationView;
use p5::{AppEvent, Context, ContextAction};
use programs::Programs;
use stacks::Stacks;
use strum::Display;
use tokio::sync::{RwLock, mpsc};

use crate::{
    model::Model,
    pulumi::{self, StackPreviewOption},
};

mod operation;
pub mod programs;
mod pulumiexec;
pub mod stacks;

#[derive(Display, Debug, Clone)]
pub enum Action {
    Event(AppEvent),
    SetContext(AppContextKey),
    MoveLeft,
    MoveRight,
    MoveUp,
    MoveDown,
    Select,
    ProgramAction(ProgramAction),
    StackAction(StackAction),
    ResourceAction(ResourceAction),
    OperationAction(OperationAction),
    PulumiAction(PulumiAction),
    VisualSelect,
}

#[derive(Display, Debug, Clone)]
pub enum PulumiAction {
    StackList(pulumi::LocalProgram),
    Output(pulumi::LocalProgram, String),
    Preview(
        pulumi::LocalProgram,
        String,
        Option<Vec<StackPreviewOption>>,
    ),
    ListStackResources(pulumi::LocalProgram, String),
    StoreStackList(Option<Vec<pulumi::Stack>>),
    StoreOutput(Option<serde_json::Value>, Option<pulumi::CommandError>),
    StorePreview(Option<pulumi::StackPreview>, Option<pulumi::CommandError>),
    StoreStackResources(Option<Vec<pulumi::StackResource>>),
}

#[derive(Display, Debug, Clone)]
pub enum OperationAction {
    Preview,
}

#[derive(Display, Debug, Clone)]
pub enum ProgramAction {
    List,
}

#[derive(Display, Debug, Clone)]
pub enum StackAction {
    Initialize,
    Refresh,
    Preview,
    Update,
    Destroy,
    Import,
    Delete,
    Rename,
    Output,
    List,
}

#[derive(Display, Debug, Clone)]
pub enum ResourceAction {
    Import,
    Delete,
    Rename,
    Refresh,
    Preview,
    Update,
    Destroy,
}

#[derive(Debug, Clone, Default)]
pub struct AppContext {
    programs: Programs,
    stacks: Stacks,
    operation_view: OperationView,
    pulumi: pulumiexec::PulumiActionHandler,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Hash)]
pub enum AppContextKey {
    #[default]
    Programs,
    Stacks,
    OperationView,
    Status,
}

#[async_trait::async_trait]
impl Context for AppContext {
    type Action = Action;
    type State = Model;

    fn handle_key_event(&self, state: &Self::State, event: Event) -> Option<Self::Action> {
        if let Event::Key(key) = event {
            match key.code {
                KeyCode::Char('h') | KeyCode::Left => {
                    return Some(Action::MoveLeft);
                }
                KeyCode::Char('l') | KeyCode::Right => {
                    return Some(Action::MoveRight);
                }
                KeyCode::Char('k') | KeyCode::Up => {
                    return Some(Action::MoveUp);
                }
                KeyCode::Char('j') | KeyCode::Down => {
                    return Some(Action::MoveDown);
                }
                KeyCode::Char(' ') | KeyCode::Enter => {
                    return Some(Action::Select);
                }
                KeyCode::Char('v') => {
                    return Some(Action::VisualSelect);
                }
                _ => {}
            }
        }

        return match state.current_context {
            AppContextKey::Programs => self.programs.handle_key_event(state, event),
            AppContextKey::Stacks => self.stacks.handle_key_event(state, event),
            AppContextKey::OperationView => self.operation_view.handle_key_event(state, event),
            AppContextKey::Status => None,
        };
    }

    async fn handle_action(
        &self,
        state: Arc<RwLock<Self::State>>,
        action: Self::Action,
        action_signal: mpsc::Sender<ContextAction<Self::Action>>,
    ) {
        match action.clone() {
            Action::Event(app_event) => {
                return state.write().await.add_event(app_event);
            }
            Action::SetContext(context) => {
                let mut state = state.write().await;
                state.current_context = context;
                return;
            }
            Action::MoveLeft => {
                let mut state = state.write().await;
                state.focus_previous_context();
                return;
            }
            Action::MoveRight => {
                let mut state = state.write().await;
                state.focus_next_context();
                return;
            }
            Action::ProgramAction(a) => {
                self.programs
                    .handle_action(state.clone(), Action::ProgramAction(a), action_signal)
                    .await;
                return;
            }
            Action::StackAction(a) => {
                self.stacks
                    .handle_action(state.clone(), Action::StackAction(a), action_signal)
                    .await;
                return;
            }
            Action::OperationAction(a) => {
                self.operation_view
                    .handle_action(state.clone(), Action::OperationAction(a), action_signal)
                    .await;
                return;
            }
            Action::PulumiAction(a) => {
                self.pulumi
                    .handle_action(state.clone(), Action::PulumiAction(a), action_signal)
                    .await;
                return;
            }
            _ => {}
        }

        let current_context = {
            let state = state.read().await;
            state.current_context.clone()
        };
        match current_context {
            AppContextKey::Programs => {
                self.programs
                    .handle_action(state.clone(), action, action_signal)
                    .await;
            }
            AppContextKey::Stacks => {
                self.stacks
                    .handle_action(state.clone(), action, action_signal)
                    .await;
            }
            AppContextKey::OperationView => {
                self.operation_view
                    .handle_action(state.clone(), action, action_signal)
                    .await;
            }
            AppContextKey::Status => {
                // Handle status context actions if needed
            }
        }
    }
}
