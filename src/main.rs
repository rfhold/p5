use crossterm::event::{Event, KeyCode, KeyEventKind, KeyModifiers};
use pui::controller::{Action, Handler, Task};
use tokio::sync::mpsc;

type Result<T> = pui::Result<T>;

#[tokio::main]
async fn main() -> Result<()> {
    let handler = AppHandler;
    let state = AppState;

    pui::run(handler, state).await
}

#[derive(Clone)]
pub struct AppHandler;

#[derive(Clone)]
pub struct AppState;

#[derive(Clone)]
pub enum AppAction {
    Exit,
}

#[derive(Clone)]
pub struct AppTask;

impl Handler for AppHandler {
    type State = AppState;
    type Action = AppAction;
    type Task = AppTask;

    #[tracing::instrument(skip(self, action_tx))]
    fn handle_event(&self, event: Event, action_tx: &mpsc::Sender<Self::Action>) -> Result<()> {
        if let Event::Key(key) = event {
            if key.kind == KeyEventKind::Press {
                let ctrl = key.modifiers.contains(KeyModifiers::CONTROL);
                match key.code {
                    KeyCode::Char('c') if ctrl => {
                        action_tx.try_send(AppAction::Exit)?;
                    }
                    _ => {}
                }
            }
        }
        Ok(())
    }
}

impl Action for AppAction {
    type State = AppState;
    type Task = AppTask;

    #[tracing::instrument(skip(self, _state, _task_tx, _action_tx))]
    fn handle_action(
        &self,
        _state: &mut Self::State,
        _task_tx: &mpsc::Sender<Self::Task>,
        _action_tx: &mpsc::Sender<Self>,
        cancel_token: &tokio_util::sync::CancellationToken,
    ) -> Result<()> {
        match self {
            AppAction::Exit => {
                tracing::info!("Exiting application...");
                cancel_token.cancel();
                Ok(())
            }
        }
    }
}

#[async_trait::async_trait]
impl Task for AppTask {
    type Action = AppAction;

    #[tracing::instrument(skip(self, _task_tx, _action_tx))]
    async fn run(
        &mut self,
        _task_tx: &mpsc::Sender<Self>,
        _action_tx: &mpsc::Sender<Self::Action>,
    ) -> Result<()> {
        Ok(())
    }
}
