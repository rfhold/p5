use pulumi_automation::{
    event::EngineEvent,
    local::LocalStack,
    stack::{PulumiProcessListener, Stack, StackChangeSummary},
};
use tokio::sync::mpsc;
use tokio_util::task::TaskTracker;

use crate::{
    actions::{AppAction, StackAction},
    state::{OperationOptions, ProgramOperation},
};

use super::AppTask;

#[derive(Clone)]
pub enum StackTask {
    RunOperation(ProgramOperation, LocalStack, OperationOptions),
}

impl OperationOptions {
    pub fn skip_preview(mut self) -> Self {
        self.skip_preview = true;
        self
    }

    pub fn preview_only(mut self) -> Self {
        self.preview_only = true;
        self
    }
}

impl StackTask {
    #[tracing::instrument(skip(self, _task_tx, action_tx))]
    pub async fn run(
        &mut self,
        _task_tx: &mpsc::Sender<AppTask>,
        action_tx: &mpsc::Sender<AppAction>,
    ) -> crate::Result<()> {
        match self {
            StackTask::RunOperation(operation, local_stack, options) => {
                let (preview_tx, mut preview_rx) = mpsc::channel::<StackChangeSummary>(100);
                let (event_tx, mut event_rx) = mpsc::channel::<EngineEvent>(100);

                let tracker = TaskTracker::new();

                let preview_stack = local_stack.clone();
                let preview_action_tx = action_tx.clone();
                let preview_operation = operation.clone();
                tracker.spawn(async move {
                    while let Some(preview) = preview_rx.recv().await {
                        if let Err(err) = preview_action_tx.try_send(AppAction::StackAction(
                            StackAction::PersistChangeSummary(
                                preview_operation.clone(),
                                preview_stack.clone(),
                                preview,
                            ),
                        )) {
                            tracing::error!("Failed to send stack update change summary: {}", err);
                        }
                    }
                });

                let event_stack = local_stack.clone();
                let event_action_tx = action_tx.clone();
                let event_operation = operation.clone();
                tracker.spawn(async move {
                    while let Some(event) = event_rx.recv().await {
                        if let Err(err) = event_action_tx.try_send(AppAction::StackAction(
                            StackAction::PersistEvent(
                                event_operation.clone(),
                                event_stack.clone(),
                                event,
                            ),
                        )) {
                            tracing::error!("Failed to send stack update event: {}", err);
                        }
                    }
                });

                match operation {
                    ProgramOperation::Update if options.preview_only => {
                        if let Err(err) = local_stack
                            .preview_async(
                                pulumi_automation::stack::StackPreviewOptions {
                                    ..Default::default()
                                },
                                PulumiProcessListener {
                                    preview_tx: Some(preview_tx),
                                    event_tx,
                                },
                            )
                            .await
                        {
                            tracing::error!("Failed to preview stack: {:?}", err);
                        }
                    }
                    ProgramOperation::Update => {
                        let mut update_options = pulumi_automation::stack::StackUpOptions {
                            ..Default::default()
                        };

                        if options.skip_preview {
                            update_options.skip_preview = Some(true);
                        }
                        if let Err(err) = local_stack
                            .up(
                                update_options,
                                PulumiProcessListener {
                                    preview_tx: Some(preview_tx),
                                    event_tx,
                                },
                            )
                            .await
                        {
                            tracing::error!("Failed to preview stack: {:?}", err);
                        }
                    }
                    ProgramOperation::Refresh => {
                        let mut refresh_options = pulumi_automation::stack::StackRefreshOptions {
                            ..Default::default()
                        };

                        if options.skip_preview {
                            refresh_options.skip_preview = Some(true);
                        }
                        if let Err(err) = local_stack
                            .refresh(
                                refresh_options,
                                PulumiProcessListener {
                                    preview_tx: Some(preview_tx),
                                    event_tx,
                                },
                            )
                            .await
                        {
                            tracing::error!("Failed to refresh stack: {:?}", err);
                        }
                    }
                    ProgramOperation::Destroy => {
                        let mut destroy_options = pulumi_automation::stack::StackDestroyOptions {
                            ..Default::default()
                        };

                        if options.skip_preview {
                            destroy_options.skip_preview = Some(true);
                        }
                        if let Err(err) = local_stack
                            .destroy(
                                destroy_options,
                                PulumiProcessListener {
                                    preview_tx: Some(preview_tx),
                                    event_tx,
                                },
                            )
                            .await
                        {
                            tracing::error!("Failed to destroy stack: {:?}", err);
                        }
                    }
                }

                tracker.close();

                tracker.wait().await;

                action_tx.try_send(AppAction::StackAction(StackAction::PersistOperationDone(
                    operation.clone(),
                    local_stack.clone(),
                )))?;

                Ok(())
            }
        }
    }
}
