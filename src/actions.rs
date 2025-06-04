use pui::controller::Action;
use pulumi_automation::{
    local::{LocalStack, LocalWorkspace},
    workspace::OutputMap,
};
use tokio::sync::mpsc;

use crate::{
    AppContext, AppState,
    state::{Loadable, StackOutputs, StackState, WorkspaceState},
    tasks::{AppTask, workspace::WorkspaceTask},
};

#[derive(Clone)]
pub enum AppAction {
    Exit,
    SubmitCommand(String),
    ToastError(String),
    PopContext,
    PushContext(AppContext),
    WorkspaceAction(WorkspaceAction),
}

#[derive(Clone)]
pub enum WorkspaceAction {
    SelectWorkspace(String),
    PersistWorkspace(LocalWorkspace),
    SelectStack(LocalWorkspace, String),
    PersistStack(LocalWorkspace, LocalStack),
    PersistStackOutputs(LocalWorkspace, LocalStack, OutputMap),
}

impl Action for AppAction {
    type State = AppState;
    type Task = AppTask;

    #[tracing::instrument(skip(self, state, task_tx, action_tx))]
    fn handle_action(
        &self,
        state: &mut Self::State,
        task_tx: &mpsc::Sender<Self::Task>,
        action_tx: &mpsc::Sender<Self>,
        cancel_token: &tokio_util::sync::CancellationToken,
    ) -> crate::Result<()> {
        match self {
            AppAction::Exit => {
                cancel_token.cancel();
                Ok(())
            }
            AppAction::PushContext(context) => {
                // TODO: Handle clearing duplicate contexts or managing context stack size
                state.context_stack.push(context.clone());
                Ok(())
            }
            AppAction::PopContext => {
                if state.context_stack.len() > 0 {
                    state.context_stack.pop();
                } else {
                    action_tx.try_send(AppAction::Exit)?;
                }
                Ok(())
            }
            AppAction::SubmitCommand(input) => {
                action_tx.try_send(crate::command::parse_command_to_action(input, state)?)?;
                state.command_prompt.reset();
                if let AppContext::CommandPrompt = state.current_context() {
                    action_tx.try_send(AppAction::PopContext)?;
                }
                Ok(())
            }
            AppAction::WorkspaceAction(action) => match action {
                WorkspaceAction::SelectWorkspace(cwd) => {
                    state.selected_workspace = Some(WorkspaceState {
                        workspace_path: cwd.clone(),
                        selected_stack: None,
                    });
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::SelectWorkspace(
                        cwd.clone(),
                    )))?;
                    Ok(())
                }
                WorkspaceAction::PersistWorkspace(workspace) => {
                    state.selected_workspace = Some(WorkspaceState {
                        workspace_path: workspace.cwd.clone(),
                        selected_stack: None,
                    });
                    state.workspaces.insert(
                        workspace.cwd.clone(),
                        crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        },
                    );
                    Ok(())
                }
                WorkspaceAction::SelectStack(workspace, name) => {
                    if state.selected_workspace.is_none()
                        || state.selected_workspace.as_ref().unwrap().workspace_path
                            != workspace.cwd
                    {
                        state.selected_workspace = Some(WorkspaceState {
                            workspace_path: workspace.cwd.clone(),
                            selected_stack: None,
                        });
                    }
                    state.selected_workspace.as_mut().unwrap().selected_stack = Some(StackState {
                        stack_name: name.clone(),
                    });
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_default()
                        .stacks
                        .insert(name.clone(), Loadable::Loading);
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::SelectStack(
                        workspace.clone(),
                        name.clone(),
                    )))?;
                    Ok(())
                }
                WorkspaceAction::PersistStack(workspace, stack) => {
                    if state.selected_workspace.is_none()
                        || state.selected_workspace.as_ref().unwrap().workspace_path
                            != workspace.cwd
                    {
                        state.selected_workspace = Some(WorkspaceState {
                            workspace_path: workspace.cwd.clone(),
                            selected_stack: None,
                        });
                    }
                    state.selected_workspace.as_mut().unwrap().selected_stack = Some(StackState {
                        stack_name: stack.name.clone(),
                    });
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_default()
                        .stacks
                        .insert(
                            stack.name.clone(),
                            Loadable::Loaded(StackOutputs {
                                stack: Loadable::Loaded(stack.clone()),
                                outputs: Loadable::Loading,
                            }),
                        );
                    Ok(())
                }
                WorkspaceAction::PersistStackOutputs(workspace, stack, outputs) => {
                    if state.selected_workspace.is_none()
                        || state.selected_workspace.as_ref().unwrap().workspace_path
                            != workspace.cwd
                    {
                        state.selected_workspace = Some(WorkspaceState {
                            workspace_path: workspace.cwd.clone(),
                            selected_stack: None,
                        });
                    }
                    if let Some(stack_state) = state
                        .selected_workspace
                        .as_mut()
                        .unwrap()
                        .selected_stack
                        .as_mut()
                    {
                        stack_state.stack_name = stack.name.clone();
                    }
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_default()
                        .stacks
                        .insert(
                            stack.name.clone(),
                            Loadable::Loaded(StackOutputs {
                                stack: Loadable::Loaded(stack.clone()),
                                outputs: Loadable::Loaded(outputs.clone()),
                            }),
                        );
                    Ok(())
                }
            },
            AppAction::ToastError(message) => {
                let expr_dt = chrono::Utc::now() + chrono::Duration::seconds(3);
                state.toast = Some((expr_dt, message.clone()));
                Ok(())
            }
        }
    }
}
