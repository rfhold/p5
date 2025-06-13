use pulumi_automation::{
    local::{LocalStack, LocalWorkspace},
    workspace::Workspace,
};
use tokio::sync::mpsc;

use crate::actions::{AppAction, WorkspaceAction};

use super::AppTask;

#[derive(Clone)]
pub enum WorkspaceTask {
    ListStacks(LocalWorkspace),
    SelectWorkspace(String),
    SelectStack(LocalWorkspace, String),
    GetStackOutputs(LocalWorkspace, LocalStack),
    GetStackConfig(LocalWorkspace, LocalStack),
    GetStackState(LocalWorkspace, LocalStack),
}

impl WorkspaceTask {
    #[tracing::instrument(skip(self, task_tx, action_tx))]
    pub async fn run(
        &mut self,
        task_tx: &mpsc::Sender<AppTask>,
        action_tx: &mpsc::Sender<AppAction>,
    ) -> crate::Result<()> {
        match self {
            WorkspaceTask::ListStacks(workspace) => {
                if let Ok(stacks) = workspace.list_stacks(Default::default()) {
                    action_tx.try_send(AppAction::WorkspaceAction(
                        WorkspaceAction::PersistStacks(workspace.clone(), stacks),
                    ))?;
                } else {
                    tracing::error!("Failed to list stacks for workspace: {}", workspace.cwd);
                }

                Ok(())
            }
            WorkspaceTask::SelectWorkspace(cwd) => {
                let workspace = LocalWorkspace::new(cwd.clone());

                action_tx.try_send(AppAction::WorkspaceAction(
                    WorkspaceAction::PersistWorkspace(workspace.clone()),
                ))?;

                Ok(())
            }
            WorkspaceTask::SelectStack(workspace, stack_name) => {
                if let Ok(stack) = workspace.select_stack(stack_name.as_str()) {
                    action_tx.try_send(AppAction::WorkspaceAction(
                        WorkspaceAction::PersistStack(workspace.clone(), stack.clone()),
                    ))?;

                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::GetStackConfig(
                        workspace.clone(),
                        stack.clone(),
                    )))?;
                } else {
                    tracing::error!("Failed to select stack: {}", stack_name);
                }

                Ok(())
            }
            WorkspaceTask::GetStackOutputs(workspace, stack) => {
                if let Ok(outputs) = workspace.stack_outputs(stack.name.as_str()) {
                    action_tx.try_send(AppAction::WorkspaceAction(
                        WorkspaceAction::PersistStackOutputs(
                            workspace.clone(),
                            stack.clone(),
                            outputs,
                        ),
                    ))?;
                } else {
                    tracing::error!("Failed to get stack outputs for stack: {}", stack.name);
                }

                Ok(())
            }
            WorkspaceTask::GetStackConfig(workspace, stack) => {
                if let Ok(config) = workspace.get_stack_config(stack.name.as_str()) {
                    action_tx.try_send(AppAction::WorkspaceAction(
                        WorkspaceAction::PersistStackConfig(
                            workspace.clone(),
                            stack.clone(),
                            config,
                        ),
                    ))?;
                } else {
                    tracing::error!("Failed to get stack config for stack: {}", stack.name);
                }

                Ok(())
            }
            WorkspaceTask::GetStackState(workspace, stack) => {
                if let Ok(state) = workspace.export_stack(stack.name.as_str()) {
                    action_tx.try_send(AppAction::WorkspaceAction(
                        WorkspaceAction::PersistStackState(workspace.clone(), stack.clone(), state),
                    ))?;
                } else {
                    tracing::error!("Failed to get stack state for stack: {}", stack.name);
                }

                Ok(())
            }
        }
    }
}
