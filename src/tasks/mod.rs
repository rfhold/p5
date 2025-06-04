use pui::controller::Task;
use tokio::sync::mpsc;

use crate::actions::AppAction;

pub mod stack;
pub mod workspace;

#[derive(Clone)]
pub enum AppTask {
    WorkspaceTask(workspace::WorkspaceTask),
}

#[async_trait::async_trait]
impl Task for AppTask {
    type Action = AppAction;

    #[tracing::instrument(skip(self, task_tx, action_tx))]
    async fn run(
        &mut self,
        task_tx: &mpsc::Sender<Self>,
        action_tx: &mpsc::Sender<Self::Action>,
    ) -> crate::Result<()> {
        match self {
            AppTask::WorkspaceTask(task) => task.run(task_tx, action_tx).await,
        }
    }
}
