use crossterm::event::{Event, KeyCode, KeyEventKind, KeyModifiers};
use pui::controller::{Action, Handler, Task};
use ratatui::widgets::{StatefulWidget, Widget};
use tokio::sync::mpsc;

type Result<T> = pui::Result<T>;

#[tokio::main]
async fn main() -> Result<()> {
    let handler = AppHandler;
    let state = AppState::default();
    let widget = App::default();

    pui::run(handler, state, widget).await
}

#[derive(Clone)]
pub struct AppHandler;

#[derive(Clone, Default)]
pub struct AppState {}

#[derive(Clone)]
pub enum AppAction {
    Select,
    Exit,
}

#[derive(Clone)]
pub enum AppTask {
    // Define any tasks needed for your application here
}

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
                    KeyCode::Char(' ') => {
                        action_tx.try_send(AppAction::Select)?;
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

    #[tracing::instrument(skip(self, state, task_tx, action_tx))]
    fn handle_action(
        &self,
        state: &mut Self::State,
        task_tx: &mpsc::Sender<Self::Task>,
        action_tx: &mpsc::Sender<Self>,
        cancel_token: &tokio_util::sync::CancellationToken,
    ) -> Result<()> {
        match self {
            AppAction::Exit => {
                tracing::info!("Exiting application...");
                cancel_token.cancel();
                Ok(())
            }
            AppAction::Select => {
                tracing::info!("Select action triggered");
                // TODO: Handle select action logic here
                Ok(())
            }
        }
    }
}

#[async_trait::async_trait]
impl Task for AppTask {
    type Action = AppAction;

    #[tracing::instrument(skip(self, task_tx, action_tx))]
    async fn run(
        &mut self,
        task_tx: &mpsc::Sender<Self>,
        action_tx: &mpsc::Sender<Self::Action>,
    ) -> Result<()> {
        Ok(())
    }
}

#[derive(Clone, Default)]
pub struct App {}

impl StatefulWidget for App {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        // Render your application UI here
        // For example, you can draw a simple text widget
        let text = "Press Ctrl-C to exit";
        let paragraph = ratatui::widgets::Paragraph::new(text)
            .block(ratatui::widgets::Block::default().title("App"))
            .alignment(ratatui::layout::Alignment::Center);

        paragraph.render(area, buf);
    }
}
