use std::sync::Arc;

use crossterm::event::{Event, KeyCode};
use p5::{Context, ContextAction};
use tokio::sync::{RwLock, mpsc};

use crate::{
    model::{Model, StackView},
    pulumi::{LocalProgram, StackPreviewOption},
};

use super::{Action, OperationAction, PulumiAction};

#[derive(Debug, Clone, Default, PartialEq, Eq, Hash)]
pub struct OperationView {}

#[async_trait::async_trait]
impl Context for OperationView {
    type Action = Action;
    type State = Model;

    fn handle_key_event(&self, _state: &Self::State, event: Event) -> Option<Self::Action> {
        if let Event::Key(key) = event {
            match key.code {
                KeyCode::Char(' ') | KeyCode::Enter => Some(Action::Select),
                KeyCode::Char('P') => Some(Action::OperationAction(OperationAction::Preview)),
                _ => None,
            }
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
                state.select_previous_operation_view_item();
            }
            Action::MoveDown => {
                let mut state = state.write().await;
                state.select_next_operation_view_item();
            }
            Action::Select => {
                let mut state = state.write().await;
                state.focus_operation_view_item();
            }
            Action::VisualSelect => {
                let mut state = state.write().await;
                state.toggle_operation_view_item();
            }
            Action::OperationAction(OperationAction::Preview) => {
                let state = state.read().await;
                if let Some(selected_program) = state.selected_program() {
                    if let Some(selected_stack) = state.selected_stack() {
                        if let StackView::Preview { steps, state, .. } = selected_stack.view {
                            let program = LocalProgram::new(
                                selected_program.name.clone(),
                                selected_program.path.clone(),
                            );

                            let targets: Vec<StackPreviewOption> = match steps {
                                Some(steps) => steps
                                    .iter()
                                    .enumerate()
                                    .filter_map(|(i, step)| {
                                        if state.is_selected(i) {
                                            Some(StackPreviewOption::Target(step.step.urn.clone()))
                                        } else {
                                            None
                                        }
                                    })
                                    .collect(),
                                None => vec![],
                            };

                            action_signal
                                .send(ContextAction::AppAction(Action::PulumiAction(
                                    PulumiAction::Preview(
                                        program.clone(),
                                        selected_stack.stack.name.clone(),
                                        Some(targets),
                                    ),
                                )))
                                .await
                                .expect("Failed to send stack preview");
                        }
                    }
                }
            }
            _ => todo!(),
        }
    }
}
