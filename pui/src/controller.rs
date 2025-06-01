use std::sync::Arc;

use crossterm::event::{Event, EventStream};
use ratatui::widgets::StatefulWidget;
use tokio::sync::Mutex;
use tokio_stream::StreamExt;
use tokio_util::task::TaskTracker;

pub trait Handler: Send + Sync + Clone + 'static {
    type State: Send + Sync + Clone + 'static;
    type Action: Action<State = Self::State, Task = Self::Task>;
    type Task: Task<Action = Self::Action>;

    fn handle_event(
        &self,
        event: Event,
        action_tx: &tokio::sync::mpsc::Sender<Self::Action>,
    ) -> color_eyre::Result<()>;
}

pub trait Action: Send + Clone + Sync + 'static {
    type State;
    type Task: Task<Action = Self>;

    fn handle_action(
        &self,
        state: &mut Self::State,
        task_tx: &tokio::sync::mpsc::Sender<Self::Task>,
        action_tx: &tokio::sync::mpsc::Sender<Self>,
        cancel_token: &tokio_util::sync::CancellationToken,
    ) -> color_eyre::Result<()>;
}

#[async_trait::async_trait]
pub trait Task: Send + Sync + Clone + 'static {
    type Action: Action<Task = Self>;

    async fn run(
        &mut self,
        task_tx: &tokio::sync::mpsc::Sender<Self>,
        action_tx: &tokio::sync::mpsc::Sender<Self::Action>,
    ) -> color_eyre::Result<()>;
}

pub struct Controller<H: Handler> {
    state: Arc<Mutex<H::State>>,
    handler: H,
    cancel_token: tokio_util::sync::CancellationToken,
}

impl<H: Handler> Controller<H> {
    pub fn new(
        handler: H,
        state: H::State,
        cancel_token: tokio_util::sync::CancellationToken,
    ) -> Self {
        Self {
            state: Arc::new(Mutex::new(state)),
            handler,
            cancel_token,
        }
    }

    #[tracing::instrument(skip(self, terminal, widget))]
    pub async fn run<W>(
        self,
        mut terminal: ratatui::DefaultTerminal,
        widget: W,
    ) -> color_eyre::Result<()>
    where
        W: StatefulWidget<State = H::State> + Send + Sync + Clone + 'static,
    {
        let mut key_events = EventStream::new();

        let (key_event_tx, mut key_event_rx) = tokio::sync::mpsc::channel::<Event>(100);
        let (action_tx, mut action_rx) = tokio::sync::mpsc::channel::<H::Action>(100);
        let (task_tx, mut task_rx) = tokio::sync::mpsc::channel::<H::Task>(100);

        let tracker = TaskTracker::new();

        tracker.spawn(async move {
            while let Some(event) = key_events.next().await {
                match event {
                    Ok(event) => {
                        if let Err(e) = key_event_tx.send(event).await {
                            tracing::error!("Failed to send key event: {}", e);
                        }
                    }
                    Err(e) => {
                        tracing::error!("Error receiving key event: {}", e);
                        continue;
                    }
                }
            }
        });

        let event_action_tx = action_tx.clone();
        tracker.spawn(async move {
            while let Some(event) = key_event_rx.recv().await {
                if let Err(e) = self.handler.handle_event(event, &event_action_tx) {
                    tracing::error!("Error handling event: {}", e);
                }
            }
        });

        let task_action_tx = action_tx.clone();
        let loopback_task_tx = task_tx.clone();
        tracker.spawn(async move {
            while let Some(mut task) = task_rx.recv().await {
                if let Err(e) = task.run(&loopback_task_tx, &task_action_tx).await {
                    tracing::error!("Error running task: {}", e);
                }
            }
        });

        let mut ticker = tokio::time::interval(std::time::Duration::from_millis(1000 / 60));

        loop {
            tokio::select! {
                action = action_rx.recv() => {
                    if let Some(action) = action {
                        let mut state = self.state.lock().await;
                        if let Err(e) = action.handle_action(&mut state, &task_tx, &action_tx, &self.cancel_token) {
                            tracing::error!("Error handling action: {}", e);
                        }
                    } else {
                        tracing::debug!("Action channel closed, exiting controller run");
                        break;
                    }
                }
                _ = ticker.tick() => {
                    let mut state = self.state.lock().await;
                    if let Err(e) = terminal.draw(|f| {
                       f.render_stateful_widget(widget.clone(), f.area() , &mut state);
                    }) {
                        tracing::error!("Error rendering terminal: {}", e);
                        break;
                    }
                }
            }
        }

        tracker.close();

        tracker.wait().await;
        tracing::debug!("All tasks completed, exiting controller run");

        Ok(())
    }
}
