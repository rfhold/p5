use std::{fmt::Display, sync::Arc, time::Duration};

use async_trait::async_trait;
use crossterm::event::{Event, EventStream, KeyCode, KeyEventKind, KeyModifiers};
use ratatui::{
    DefaultTerminal,
    prelude::*,
    widgets::{ListItem, StatefulWidget},
};

use tokio::sync::{RwLock, mpsc, watch};
use tokio_stream::StreamExt;

#[derive(Debug, Clone)]
pub struct AppEvent {
    pub message: String,
    pub timestamp: String,
    pub event_type: EventType,
    pub source: String,
    pub command_result: Option<CommandResult>,
}

impl Display for AppEvent {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}: {}", self.timestamp, self.message)
    }
}

impl<'a> Into<ListItem<'a>> for AppEvent {
    fn into(self) -> ListItem<'a> {
        let timestamp = self.timestamp.clone();
        let message = self.message.clone();
        let timestamp_text = format!("{}: ", timestamp);

        return ListItem::new(Line::from(vec![
            timestamp_text.dark_gray(),
            " ".into(),
            Span::from(message).style(match self.event_type {
                EventType::Error => Style::default().fg(Color::Red),
                EventType::Destructive => Style::default().fg(Color::Rgb(255, 165, 0)),
                EventType::Info => Style::default().fg(Color::Green),
            }),
        ]));
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum EventType {
    Error,
    Destructive,
    Info,
}

#[derive(Debug, Clone)]
pub struct CommandResult {
    pub stdout: String,
    pub stderr: String,
    pub code: i32,
}

#[derive(Debug, Clone)]
pub enum ContextAction<A> {
    Stop,
    AppAction(A),
}

#[async_trait]
pub trait Context: Sync + Send + Clone {
    type Action: Clone + Send + Sync;
    type State: Clone + Send + Sync;

    fn handle_key_event(&self, state: &Self::State, event: Event) -> Option<Self::Action>;
    async fn handle_action(
        &self,
        state: Arc<RwLock<Self::State>>,
        action: Self::Action,
        action_signal: mpsc::Sender<ContextAction<Self::Action>>,
    );
}

pub struct ContextController<S, A> {
    state: Arc<RwLock<S>>,
    fps: f32,
    running_state: RunningState,
    boot_actions: Vec<ContextAction<A>>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
enum RunningState {
    Default,
    Done,
}

impl<'a: 'static, S: Sync + Send + Clone + 'a, A: Send + Sync + Clone + 'a + std::fmt::Debug>
    ContextController<S, A>
{
    pub fn new(state: S) -> Self {
        Self {
            state: Arc::new(RwLock::new(state)),
            running_state: RunningState::Default,
            fps: 60.0,
            boot_actions: Vec::new(),
        }
    }

    pub fn with_boot_actions(mut self, actions: Vec<ContextAction<A>>) -> Self {
        self.boot_actions = actions;
        self
    }

    pub async fn run<C, W: StatefulWidget<State = S> + Clone>(
        self,
        mut terminal: DefaultTerminal,
        context: C,
        app: W,
    ) -> color_eyre::Result<()>
    where
        C: Context<Action = A, State = S> + Send + Sync + 'static,
    {
        let (action_queue, action_receiver) = mpsc::channel::<ContextAction<A>>(10);

        let (shutdown_tx, shutdown_rx) = watch::channel(());

        let mut handles: Vec<tokio::task::JoinHandle<()>> = Vec::new();

        for action in self.boot_actions.clone() {
            action_queue.send(action).await.unwrap();
        }

        let state = self.state.clone();
        let key_action_queue = action_queue.clone();
        handles.push(tokio::spawn(Self::key_task(
            key_action_queue,
            state.clone(),
            context.clone(),
            shutdown_rx.clone(),
        )));

        let controller = Arc::new(RwLock::new(self));
        let action_controller = controller.clone();
        handles.push(tokio::spawn(Self::action_task(
            action_receiver,
            action_queue.clone(),
            action_controller,
            state.clone(),
            context.clone(),
            shutdown_rx.clone(),
        )));

        let period = {
            let rw_lock = controller.clone();
            let controller = rw_lock.read().await;
            Duration::from_secs_f32(1.0 / controller.fps)
        };
        let mut interval = tokio::time::interval(period);

        loop {
            let controller = controller.read().await;

            if RunningState::Done == controller.running_state {
                break;
            }

            tokio::select! {
                _ = interval.tick() => {
                    let mut state = controller.state.write().await;
                    terminal.draw(|frame| frame.render_stateful_widget(app.clone(), frame.area(), &mut state));
                },
            }
        }

        let _ = shutdown_tx.send(());

        for handle in handles {
            let _ = handle.await;
        }

        Ok(())
    }

    async fn key_task<C>(
        action_queue: mpsc::Sender<ContextAction<A>>,
        state: Arc<RwLock<S>>,
        context: C,
        mut shutdown_rx: watch::Receiver<()>,
    ) where
        C: Context<Action = A, State = S> + Send + Sync + 'static,
    {
        let mut key_events = EventStream::new();

        loop {
            tokio::select! {
                Some(Ok(event)) = key_events.next() => {
                    // Self::handle_key_event(action_queue.clone(), controller.clone(), &event).await;
                    if let Event::Key(key) = event {
                        if key.kind == KeyEventKind::Press {
                            let ctrl = key.modifiers.contains(KeyModifiers::CONTROL);
                            match key.code {
                                KeyCode::Char('c') if ctrl => {
                                    action_queue.send(ContextAction::Stop).await.unwrap();
                                    return;
                                }
                                _ => {}
                            }
                        }
                    }

                    if let Some(action) = context.handle_key_event(&state.read().await.clone(), event) {
                        action_queue
                            .send(ContextAction::AppAction(
                                action.clone(),
                            ))
                            .await
                            .expect("Failed to send action signal");
                    }
                }
                _ = shutdown_rx.changed() => {
                    break;
                }
            }
        }
    }

    async fn action_task<C>(
        mut action_receiver: mpsc::Receiver<ContextAction<A>>,
        action_queue: mpsc::Sender<ContextAction<A>>,
        controller: Arc<RwLock<Self>>,
        state: Arc<RwLock<S>>,
        context: C,
        mut shutdown_rx: watch::Receiver<()>,
    ) where
        C: Context<Action = A, State = S> + Send + Sync + 'static,
    {
        loop {
            tokio::select! {
                Some(action) = action_receiver.recv() => {
                    match action {
                        ContextAction::Stop => {
                            let mut controller = controller.write().await;
                            controller.running_state = RunningState::Done;
                        }
                        ContextAction::AppAction(action) => {
                            let state_clone = state.clone();
                            let action_queue_clone = action_queue.clone();
                            let context_clone = context.clone();
                            tokio::spawn(async move {
                                context_clone.handle_action(
                                    state_clone,
                                    action,
                                    action_queue_clone,
                                ).await;
                            });
                        }
                    }
                }
                _ = shutdown_rx.changed() => {
                    break;
                }
            }
        }
    }
}
