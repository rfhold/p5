use std::sync::Arc;

use crossterm::event::{Event, EventStream};
use ratatui::{Terminal, widgets::StatefulWidget};
use tokio::sync::Mutex;
use tokio_stream::StreamExt;
use tokio_util::task::TaskTracker;

pub trait Handler: Send + Sync + Clone + 'static {
    type State: Send + Sync + 'static;
    type Action: Action<State = Self::State, Task = Self::Task>;
    type Task: Task<Action = Self::Action>;

    fn handle_event(
        &self,
        state: &mut Self::State,
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
    pub async fn run<W, B>(self, mut terminal: Terminal<B>, widget: W) -> color_eyre::Result<()>
    where
        W: StatefulWidget<State = H::State> + Send + Sync + Clone + 'static,
        B: ratatui::backend::Backend + Send + Sync + 'static,
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
        let event_state = self.state.clone();
        tracker.spawn(async move {
            while let Some(event) = key_event_rx.recv().await {
                let mut state = event_state.lock().await;
                if let Err(e) = self
                    .handler
                    .handle_event(&mut state, event, &event_action_tx)
                {
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

#[cfg(test)]
mod tests {
    use ratatui::backend::TestBackend;

    use super::*;

    #[derive(Clone, Default)]
    struct TestHandler;

    #[derive(Clone)]
    enum TestAction {
        TestAction(String),
        StartTask(String, Vec<TestAction>),
        RecordTask(String),
        RecordKeyEvent(crossterm::event::KeyEvent),
    }

    #[derive(Clone)]
    enum TestTask {
        RecordTask(String, Vec<TestAction>),
    }

    #[derive(Clone, Default)]
    struct TestState {
        actions: Vec<String>,
        tasks: Vec<String>,
        key_events: Vec<crossterm::event::KeyEvent>,
    }

    #[derive(Clone, Default)]
    struct TestApp {}

    impl StatefulWidget for TestApp {
        type State = TestState;

        fn render(
            self,
            area: ratatui::layout::Rect,
            buf: &mut ratatui::buffer::Buffer,
            _state: &mut Self::State,
        ) {
            // Render logic for the test app
            buf.set_string(area.x, area.y, "Test App", ratatui::style::Style::default());
        }
    }

    impl Handler for TestHandler {
        type State = TestState;
        type Action = TestAction;
        type Task = TestTask;

        fn handle_event(
            &self,
            _state: &mut Self::State,
            event: Event,
            action_tx: &tokio::sync::mpsc::Sender<Self::Action>,
        ) -> color_eyre::Result<()> {
            if let Event::Key(key_event) = event {
                if key_event.kind == crossterm::event::KeyEventKind::Press {
                    action_tx.try_send(TestAction::RecordKeyEvent(key_event))?;
                    match key_event.code {
                        crossterm::event::KeyCode::Char('t')
                            if key_event
                                .modifiers
                                .contains(crossterm::event::KeyModifiers::CONTROL) =>
                        {
                            action_tx
                                .try_send(TestAction::TestAction("Ctrl-T pressed".to_string()))?;
                        }
                        crossterm::event::KeyCode::Char('t')
                            if key_event
                                .modifiers
                                .contains(crossterm::event::KeyModifiers::SHIFT) =>
                        {
                            action_tx.try_send(TestAction::StartTask(
                                "TestTast2".to_string(),
                                vec![
                                    TestAction::RecordTask("Sub2Task1".to_string()),
                                    TestAction::RecordTask("Sub2Task2".to_string()),
                                    TestAction::RecordTask("Sub2Task3".to_string()),
                                ],
                            ))?;
                        }
                        crossterm::event::KeyCode::Char('t') => {
                            action_tx.try_send(TestAction::StartTask(
                                "TestTask".to_string(),
                                vec![TestAction::RecordTask("SubTask1".to_string())],
                            ))?;
                        }
                        _ => {}
                    }
                }
            }
            Ok(())
        }
    }

    impl Action for TestAction {
        type State = TestState;
        type Task = TestTask;

        fn handle_action(
            &self,
            state: &mut Self::State,
            task_tx: &tokio::sync::mpsc::Sender<Self::Task>,
            _action_tx: &tokio::sync::mpsc::Sender<Self>,
            _cancel_token: &tokio_util::sync::CancellationToken,
        ) -> color_eyre::Result<()> {
            match self {
                TestAction::TestAction(msg) => {
                    state.actions.push(msg.clone());
                }
                TestAction::RecordTask(task_name) => {
                    state.tasks.push(task_name.clone());
                }
                TestAction::RecordKeyEvent(key_event) => {
                    state.key_events.push(*key_event);
                }
                TestAction::StartTask(task_name, actions) => {
                    state.tasks.push(task_name.clone());
                    task_tx.try_send(TestTask::RecordTask(task_name.clone(), actions.clone()))?;
                }
            }
            Ok(())
        }
    }

    #[async_trait::async_trait]
    impl Task for TestTask {
        type Action = TestAction;

        async fn run(
            &mut self,
            _task_tx: &tokio::sync::mpsc::Sender<Self>,
            action_tx: &tokio::sync::mpsc::Sender<Self::Action>,
        ) -> color_eyre::Result<()> {
            match self {
                TestTask::RecordTask(task_name, actions) => {
                    tracing::info!("Running task: {}", task_name);
                    for action in actions {
                        action_tx.try_send(action.clone())?;
                    }
                }
            }
            Ok(())
        }
    }

    #[tokio::test]
    async fn test_start_and_stop_controller() {
        let cancel_token = tokio_util::sync::CancellationToken::new();
        let handler = TestHandler;
        let state = TestState::default();

        let controller = Controller::new(handler, state, cancel_token.clone());

        let start_time = std::time::Instant::now();

        let terminal = ratatui::Terminal::new(TestBackend::new(80, 30)).unwrap();

        let handle_cancel_token = cancel_token.clone();
        let handle = tokio::spawn(async move {
            tokio::select! {
                res = controller.run(terminal, TestApp::default()) => {
                    assert!(res.is_ok(), "Controller run failed: {:?}", res);
                }
                _ = handle_cancel_token.cancelled() => {
                    tracing::info!("Cancellation token triggered, shutting down...");
                }
            }
        });

        tokio::time::sleep(std::time::Duration::from_millis(333)).await;

        cancel_token.cancel();
        handle.await.unwrap();

        let elapsed = start_time.elapsed();

        assert!(
            elapsed.as_millis() - 333 < 10,
            "Controller was not did not stop within expected time: {:?} ms",
            elapsed
        );
    }
}
