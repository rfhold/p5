use actions::AppAction;
use crossterm::event::{Event, KeyCode, KeyEventKind, KeyModifiers};
use p5::controller::Handler;
use state::{AppContext, AppState};
use tasks::AppTask;
use tokio::sync::mpsc;

mod actions;
mod command;
mod input;
mod layout;
mod state;
mod tasks;
mod widgets;

pub(crate) type Result<T> = p5::Result<T>;

#[tokio::main]
async fn main() -> Result<()> {
    let handler = AppHandler;
    let state = AppState {
        context_stack: vec![AppContext::WorkspaceList],
        ..Default::default()
    };

    let init_actions = vec![AppAction::ListWorkspaces];

    p5::run(handler, state, init_actions, layout::AppLayout::default()).await
}

#[derive(Clone)]
pub struct AppHandler;

impl Handler for AppHandler {
    type State = AppState;
    type Action = AppAction;
    type Task = AppTask;

    #[tracing::instrument(skip(self, state, action_tx))]
    fn handle_event(
        &self,
        state: &mut Self::State,
        event: Event,
        action_tx: &mpsc::Sender<Self::Action>,
    ) -> Result<()> {
        if let AppContext::CommandPrompt = state.current_context() {
            if let Some(input) = input::handle_user_input(event, &mut state.command_prompt) {
                match input {
                    input::InputResult::Submit(command) => {
                        tracing::info!("Command submitted: {}", command);
                        action_tx.try_send(AppAction::SubmitCommand(command.clone()))?;
                    }
                    input::InputResult::Cancel => {
                        action_tx.try_send(AppAction::PopContext)?;
                    }
                }
            }
        } else if let Event::Key(key) = event {
            if key.kind == KeyEventKind::Press {
                match key.code {
                    KeyCode::Char('j') | KeyCode::Down => {
                        action_tx.try_send(AppAction::NavigateDown)?;
                    }
                    KeyCode::Char('k') | KeyCode::Up => {
                        action_tx.try_send(AppAction::NavigateUp)?;
                    }
                    KeyCode::Char('h') | KeyCode::Left => {
                        action_tx.try_send(AppAction::NavigateLeft)?;
                    }
                    KeyCode::Char('l') | KeyCode::Right => {
                        action_tx.try_send(AppAction::NavigateRight)?;
                    }
                    KeyCode::Char(':') => {
                        action_tx.try_send(AppAction::PushContext(AppContext::CommandPrompt))?;
                    }
                    KeyCode::Esc => {
                        action_tx.try_send(AppAction::PopContext)?;
                    }
                    KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                        action_tx.try_send(AppAction::Exit)?;
                    }
                    KeyCode::Char(' ') => {}
                    _ => {}
                }
            }
        }
        Ok(())
    }
}
