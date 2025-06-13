use glob::glob;
use p5::controller::Task;
use pulumi_automation::local::LocalWorkspace;
use tokio::sync::mpsc;

use crate::actions::AppAction;

pub mod stack;
pub mod workspace;

#[derive(Clone)]
pub enum AppTask {
    WorkspaceTask(workspace::WorkspaceTask),
    StackTask(stack::StackTask),
    ListWorkspaces,
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
            AppTask::StackTask(task) => task.run(task_tx, action_tx).await,
            AppTask::ListWorkspaces => {
                let mut workspaces = Vec::new();

                for entry in glob("**/Pulumi.yaml")? {
                    match entry {
                        Ok(path) => workspaces.push(LocalWorkspace::new(
                            path.parent().unwrap().to_string_lossy().to_string(),
                        )),
                        Err(e) => tracing::error!("Error reading glob entry: {}", e),
                    }
                }

                action_tx.try_send(AppAction::PersistWorkspaces(workspaces))?;

                Ok(())
            }
        }
    }
}
