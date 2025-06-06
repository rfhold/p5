use pulumi_automation::{
    local::LocalStack,
    stack::{Stack, StackPreviewOptions},
};
use tokio::sync::mpsc;

use crate::actions::{AppAction, StackAction};

use super::AppTask;

#[derive(Clone)]
pub enum StackTask {
    GetStackPreview(LocalStack),
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
        }
    }
}
