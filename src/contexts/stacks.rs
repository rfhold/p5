use std::sync::Arc;

use crossterm::event::{Event, KeyCode};
use p5::{AppEvent, Context, ContextAction, EventType};
use tokio::sync::{RwLock, mpsc};

use std::default::Default;

use crate::{
    model::{Model, StackList, StackModel},
    pulumi::{self, Interactor, LocalProgram, StackOutputOption, StackPreviewOption},
};

use super::{Action, AppContextKey, PulumiAction, ResourceAction, StackAction};

#[derive(Debug, Clone, Default, PartialEq, Eq, Hash)]
pub struct Stacks {}

#[async_trait::async_trait]
impl Context for Stacks {
    type Action = Action;
    type State = Model;

    fn handle_key_event(&self, state: &Self::State, event: Event) -> Option<Self::Action> {
        if let Event::Key(key) = event {
            return match key.code {
                KeyCode::Char('o') => Some(Action::StackAction(StackAction::Output)),
                KeyCode::Char('P') => Some(Action::StackAction(StackAction::Preview)),
                _ => None,
            };
        } else {
            None
        }
    }

    async fn handle_action(
        &self,
        state: Arc<RwLock<Self::State>>,
        action: Self::Action,
        action_signal: mpsc::Sender<ContextAction<Self::Action>>,
    ) {
        match action {
            Action::MoveUp => {
                let mut state = state.write().await;
                state.select_previous_stack();
            }
            Action::MoveDown => {
                let mut state = state.write().await;
                state.select_next_stack();
            }
            Action::Select => {
                let mut state = state.write().await;

                if let Some(selected_program) = state.selected_program() {
                    if let Some(selected) = state.select_highlighted_stack() {
                        action_signal
                            .send(ContextAction::AppAction(Action::PulumiAction(
                                PulumiAction::ListStackResources(
                                    pulumi::LocalProgram::new(
                                        selected_program.name.clone(),
                                        selected_program.path.clone(),
                                    ),
                                    selected.stack.name.clone(),
                                ),
                            )))
                            .await
                            .expect("Failed to send resource action");
                        action_signal
                            .send(ContextAction::AppAction(Action::SetContext(
                                AppContextKey::OperationView,
                            )))
                            .await
                            .expect("Failed to send context change signal");
                    }
                }
            }
            Action::StackAction(StackAction::List) => {
                let state = state.read().await;
                if let Some(selected_program) = state.selected_program() {
                    let program = LocalProgram::new(
                        selected_program.name.clone(),
                        selected_program.path.clone(),
                    );

                    action_signal
                        .send(ContextAction::AppAction(Action::PulumiAction(
                            PulumiAction::StackList(program.clone()),
                        )))
                        .await
                        .expect("Failed to send stack list action");
                }
            }
            Action::StackAction(StackAction::Output) => {
                let mut state = state.write().await;
                if let Some(selected_program) = &mut state.selected_program() {
                    if let Some(selected_stack) = &mut state.select_highlighted_stack() {
                        let program = LocalProgram::new(
                            selected_program.name.clone(),
                            selected_program.path.clone(),
                        );

                        action_signal
                            .send(ContextAction::AppAction(Action::PulumiAction(
                                PulumiAction::Output(
                                    program.clone(),
                                    selected_stack.stack.name.clone(),
                                ),
                            )))
                            .await
                            .expect("Failed to send stack output");
                    }
                }
            }
            Action::StackAction(StackAction::Preview) => {
                let mut state = state.write().await;
                if let Some(selected_program) = &mut state.selected_program() {
                    if let Some(selected_stack) = &mut state.select_highlighted_stack() {
                        let program = LocalProgram::new(
                            selected_program.name.clone(),
                            selected_program.path.clone(),
                        );

                        action_signal
                            .send(ContextAction::AppAction(Action::PulumiAction(
                                PulumiAction::Preview(
                                    program.clone(),
                                    selected_stack.stack.name.clone(),
                                    Some(vec![StackPreviewOption::ShowReplacementSteps]),
                                ),
                            )))
                            .await
                            .expect("Failed to send stack preview");
                    }
                }
            }
            _ => todo!(),
        }
    }
}
