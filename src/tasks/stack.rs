use pulumi_automation::{
    event::EngineEvent,
    local::LocalStack,
    stack::{PulumiProcessListener, Stack, StackChangeSummary, StackPreviewOptions},
};
use tokio::sync::mpsc;
use tokio_util::task::TaskTracker;

use crate::actions::{AppAction, StackAction};

use super::AppTask;

#[derive(Clone)]
pub enum StackTask {
    GetStackPreview(LocalStack),
    UpdateStack(LocalStack, StackUpdateOptions),
}

#[derive(Clone, Default)]
pub struct StackUpdateOptions {
    pub skip_preview: bool,
}

impl StackTask {
    #[tracing::instrument(skip(self, _task_tx, action_tx))]
    pub async fn run(
        &mut self,
        _task_tx: &mpsc::Sender<AppTask>,
        action_tx: &mpsc::Sender<AppAction>,
    ) -> crate::Result<()> {
        match self {
            StackTask::GetStackPreview(local_stack) => {
                if let Ok(preview) = local_stack.preview(StackPreviewOptions::default()) {
                    action_tx.try_send(AppAction::StackAction(
                        StackAction::PersistStackPreview(local_stack.clone(), preview),
                    ))?;
                } else {
                    tracing::error!("Failed to preview stack: {}", local_stack.name);
                }

                Ok(())
            }
            StackTask::UpdateStack(local_stack, options) => {
                let (preview_tx, mut preview_rx) = mpsc::channel::<StackChangeSummary>(100);
                let (event_tx, mut event_rx) = mpsc::channel::<EngineEvent>(100);

                let tracker = TaskTracker::new();

                let preview_stack = local_stack.clone();
                let preview_action_tx = action_tx.clone();
                tracker.spawn(async move {
                    while let Some(preview) = preview_rx.recv().await {
                        if let Err(err) = preview_action_tx.try_send(AppAction::StackAction(
                            StackAction::PersistStackUpdateChangeSummary(
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
                tracker.spawn(async move {
                    while let Some(event) = event_rx.recv().await {
                        if let Err(err) = event_action_tx.try_send(AppAction::StackAction(
                            StackAction::PersistStackUpdateEvent(event_stack.clone(), event),
                        )) {
                            tracing::error!("Failed to send stack update event: {}", err);
                        }
                    }
                });

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
                    tracing::error!("Failed to update stack: {:?}", err);
                }

                tracker.close();

                tracker.wait().await;

                action_tx.try_send(AppAction::StackAction(StackAction::PersistStackUpdateDone(
                    local_stack.clone(),
                )))?;

                Ok(())
            }
        }
    }
}
