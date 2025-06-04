use pulumi_automation::{local::LocalWorkspace, workspace::Workspace};
use tokio::sync::mpsc;

use crate::actions::{AppAction, WorkspaceAction};

use super::AppTask;

#[derive(Clone)]
pub enum WorkspaceTask {
    SelectWorkspace(String),
    SelectStack(LocalWorkspace, String),
}

impl WorkspaceTask {
    #[tracing::instrument(skip(self, _task_tx, action_tx))]
    pub async fn run(
        &mut self,
        _task_tx: &mpsc::Sender<AppTask>,
        action_tx: &mpsc::Sender<AppAction>,
    ) -> crate::Result<()> {
        match self {
            WorkspaceTask::SelectWorkspace(cwd) => {
                let workspace = LocalWorkspace::new(cwd.clone());

                action_tx.try_send(AppAction::WorkspaceAction(
                    WorkspaceAction::PersistWorkspace(workspace.clone()),
                ))?;

                Ok(())
            }
            WorkspaceTask::SelectStack(workspace, stack_name) => {
                let stack = workspace.select_stack(stack_name.as_str()).unwrap();

                action_tx.try_send(AppAction::WorkspaceAction(WorkspaceAction::PersistStack(
                    workspace.clone(),
                    stack.clone(),
                )))?;

                if let Ok(outputs) = workspace.stack_outputs(stack_name.as_str()) {
                    action_tx.try_send(AppAction::WorkspaceAction(
                        WorkspaceAction::PersistStackOutputs(
                            workspace.clone(),
                            stack.clone(),
                            outputs,
                        ),
                    ))?;
                } else {
                    tracing::error!("Failed to get stack outputs for stack: {}", stack_name);
                }

                Ok(())
            }
        }
    }
}
