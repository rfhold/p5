use crossterm::event::{Event, EventStream, KeyCode, KeyEventKind, KeyModifiers};
use tokio_stream::StreamExt;
use tokio_util::task::TaskTracker;

pub trait Action: Send + Clone + Sync + 'static {
    type State;
    type Task: Task<Action = Self>;

    fn handle_action(
        &self,
        state: &mut Self::State,
        task_tx: &tokio::sync::mpsc::Sender<Self::Task>,
        action_tx: &tokio::sync::mpsc::Sender<Self>,
    ) -> color_eyre::Result<()>;
}

pub trait Task: Send + Sync + Clone + 'static {
    type Action: Action<Task = Self>;

    async fn run(
        &mut self,
        task_tx: &tokio::sync::mpsc::Sender<Self>,
        action_tx: &tokio::sync::mpsc::Sender<Self::Action>,
    ) -> color_eyre::Result<()>;
}

pub struct Controller {
    cancel_token: tokio_util::sync::CancellationToken,
}

impl Controller {
    pub fn new(cancel_token: tokio_util::sync::CancellationToken) -> Self {
        Self { cancel_token }
    }

    #[tracing::instrument(skip(self, _terminal))]
    pub async fn run(self, _terminal: ratatui::DefaultTerminal) -> color_eyre::Result<()> {
        let mut key_events = EventStream::new();

        let tracker = TaskTracker::new();

        tracker.spawn(async move {
            loop {
                tokio::select! {
                    _ = self.cancel_token.cancelled() => {
                        tracing::info!("Breaking out of event loop due to cancellation token");
                        break;
                    },
                    Some(Ok(event)) = key_events.next() => {
                        if let Event::Key(key) = event {
                            if key.kind == KeyEventKind::Press {
                                tracing::trace!("Key pressed: {:?}", key);

                                match key.code {
                                    KeyCode::Char('q') => {
                                        self.cancel_token.cancel();
                                    }
                                    KeyCode::Char('c') if key.modifiers == KeyModifiers::CONTROL => {
                                        self.cancel_token.cancel();
                                    }
                                    _ => {
                                        tracing::debug!("Unhandled key event: {:?}", key);
                                    }
                                }
                            }
                        }
                    },
                }
            }
        });

        tracker.close();

        tracker.wait().await;
        tracing::debug!("All tasks completed, exiting controller run");

        Ok(())
    }
}

#[cfg(test)]
mod test {}
