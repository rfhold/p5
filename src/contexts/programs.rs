use std::sync::Arc;

use crossterm::event::{Event, KeyCode};
use p5::{AppEvent, Context, ContextAction, EventType};
use tokio::sync::{RwLock, mpsc};

use crate::{
    model::{Model, Program},
    pulumi::{self},
};

use super::{Action, AppContextKey, ProgramAction, StackAction};

#[derive(Debug, Clone, Default, PartialEq, Eq, Hash)]
pub struct Programs {}

#[async_trait::async_trait]
impl Context for Programs {
    type Action = Action;
    type State = Model;

    fn handle_key_event(&self, state: &Self::State, event: Event) -> Option<Self::Action> {
        if let Event::Key(key) = event {
            match key.code {
                KeyCode::Char(' ') | KeyCode::Enter => Some(Action::Select),
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
                state.select_previous_program();
            }
            Action::MoveDown => {
                let mut state = state.write().await;
                state.select_next_program();
            }
            Action::Select => {
                let mut state = state.write().await;
                if let Some(_selected) = state.select_highlighted_program() {
                    action_signal
                        .send(ContextAction::AppAction(Action::StackAction(
                            StackAction::List,
                        )))
                        .await
                        .expect("Failed to send stack action");
                    action_signal
                        .send(ContextAction::AppAction(Action::SetContext(
                            AppContextKey::Stacks,
                        )))
                        .await
                        .expect("Failed to send context change signal");
                }
            }
            Action::ProgramAction(ProgramAction::List) => {
                let path = match std::env::current_dir() {
                    Ok(path) => path,
                    Err(err) => {
                        action_signal
                            .send(ContextAction::AppAction(Action::Event(AppEvent {
                                message: format!("Error getting current directory: {}", err),
                                timestamp: chrono::Utc::now().to_string(),
                                event_type: EventType::Error,
                                source: "PulumiTask".to_string(),
                                command_result: None,
                            })))
                            .await
                            .expect("Failed to send error event");
                        return;
                    }
                };

                let new_programs = match pulumi::LocalProgram::all_in_cwd(path, 2) {
                    Ok(new_programs) => new_programs,
                    Err(error) => {
                        // TODO: Handle error
                        // action_signal
                        //     .send(ContextAction::AppAction(Action::Event(AppEvent {
                        //         message: format!("Error finding programs: {}", error),
                        //         timestamp: chrono::Utc::now().to_string(),
                        //         event_type: EventType::Error,
                        //         source: "PulumiTask".to_string(),
                        //         command_result: None,
                        //     })))
                        //     .await
                        //     .expect("Failed to send error event");
                        return;
                    }
                };

                // Update the state with the merged programs
                let mut state = state.write().await;

                state.program_list.programs = new_programs
                    .iter()
                    .map(|p| Program {
                        name: p.name.clone(),
                        path: p.cwd.clone(),
                        selected_stack: None,
                        stack_list: None,
                    })
                    .collect();
            }
            _ => todo!(),
        }
    }
}
