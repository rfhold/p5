use pui::controller::Action;
use pulumi_automation::{
    local::{LocalStack, LocalWorkspace},
    workspace::{Deployment, OutputMap, StackSettings},
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
    PersistStackConfig(LocalWorkspace, LocalStack, StackSettings),
    PersistStackState(LocalWorkspace, LocalStack, Deployment),
    LoadStackState(LocalWorkspace, LocalStack),
    LoadStackOutputs(LocalWorkspace, LocalStack),
    LoadStackConfig(LocalWorkspace, LocalStack),
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

                match context {
                    AppContext::Stack(stack_context) => {
                        let workspace = state.workspace();
                        let stack = state.stack();

                        if let (Loadable::Loaded(workspace), Loadable::Loaded(stack)) =
                            (workspace, stack)
                        {
                            match stack_context {
                                crate::state::StackContext::Outputs => {
                                    action_tx.try_send(AppAction::WorkspaceAction(
                                        WorkspaceAction::LoadStackOutputs(
                                            workspace.clone(),
                                            stack.clone(),
                                        ),
                                    ))?;
                                }
                                crate::state::StackContext::Config => {
                                    action_tx.try_send(AppAction::WorkspaceAction(
                                        WorkspaceAction::LoadStackConfig(
                                            workspace.clone(),
                                            stack.clone(),
                                        ),
                                    ))?;
                                }
                                crate::state::StackContext::Resources => {
                                    action_tx.try_send(AppAction::WorkspaceAction(
                                        WorkspaceAction::LoadStackState(
                                            workspace.clone(),
                                            stack.clone(),
                                        ),
                                    ))?;
                                }
                            }
                        }
                    }
                    _ => {}
                }

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
                if let Some(action) = crate::command::parse_command_to_action(input, state)? {
                    action_tx.try_send(action)?;
                }
                state.command_prompt.reset();
                if let AppContext::CommandPrompt = state.current_context() {
                    if state.context_stack.len() > 1 {
                        // If we are in the command prompt context, pop it to return to the previous context
                        state.context_stack.pop();
                    }
                }
                Ok(())
            }
            AppAction::WorkspaceAction(action) => match action {
                WorkspaceAction::SelectWorkspace(cwd) => {
                    state.selected_workspace = Some(WorkspaceState {
                        workspace_path: cwd.clone(),
                        selected_stack: None,
                    });
                    state.workspaces.entry(cwd.clone()).or_insert_with(|| {
                        crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loading,
                            stacks: Default::default(),
                        }
                    });
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::SelectWorkspace(
                        cwd.clone(),
                    )))?;
                    Ok(())
                }
                WorkspaceAction::PersistWorkspace(workspace) => {
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .and_modify(|w| {
                            if let Loadable::Loading = w.workspace {
                                *w = crate::state::WorkspaceOutputs {
                                    workspace: Loadable::Loaded(workspace.clone()),
                                    stacks: Default::default(),
                                };
                            }
                        })
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::SelectStack(workspace, name) => {
                    state
                        .selected_workspace
                        .get_or_insert_with(|| WorkspaceState {
                            workspace_path: workspace.cwd.clone(),
                            selected_stack: None,
                        })
                        .selected_stack = Some(StackState {
                        stack_name: name.clone(),
                    });
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        })
                        .stacks
                        .entry(name.clone())
                        .and_modify(|s| {
                            s.stack = Loadable::Loading;
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loading,
                            config: Loadable::Loading,
                            outputs: Default::default(),
                            state: Default::default(),
                        });
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::SelectStack(
                        workspace.clone(),
                        name.clone(),
                    )))?;
                    action_tx.try_send(AppAction::PushContext(AppContext::Stack(
                        Default::default(),
                    )))?;
                    Ok(())
                }
                WorkspaceAction::PersistStack(workspace, stack) => {
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        })
                        .stacks
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.stack = Loadable::Loaded(stack.clone());
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Loadable::Loading,
                            outputs: Default::default(),
                            state: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::PersistStackOutputs(workspace, stack, outputs) => {
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        })
                        .stacks
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.outputs = Loadable::Loaded(outputs.clone());
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            outputs: Loadable::Loaded(outputs.clone()),
                            config: Default::default(),
                            state: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::PersistStackConfig(workspace, stack, config) => {
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        })
                        .stacks
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.config = Loadable::Loaded(config.clone());
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Loadable::Loaded(config.clone()),
                            outputs: Default::default(),
                            state: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::PersistStackState(workspace, stack, stack_state) => {
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        })
                        .stacks
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.state = Loadable::Loaded(stack_state.clone());
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Default::default(),
                            outputs: Default::default(),
                            state: Loadable::Loaded(stack_state.clone()),
                        });
                    Ok(())
                }
                WorkspaceAction::LoadStackState(workspace, stack) => {
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::GetStackState(
                        workspace.clone(),
                        stack.clone(),
                    )))?;
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        })
                        .stacks
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.state = Loadable::Loading;
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Default::default(),
                            outputs: Default::default(),
                            state: Loadable::Loading,
                        });
                    Ok(())
                }
                WorkspaceAction::LoadStackOutputs(workspace, stack) => {
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::GetStackOutputs(
                        workspace.clone(),
                        stack.clone(),
                    )))?;
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        })
                        .stacks
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.outputs = Loadable::Loading;
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Default::default(),
                            outputs: Loadable::Loading,
                            state: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::LoadStackConfig(workspace, stack) => {
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::GetStackConfig(
                        workspace.clone(),
                        stack.clone(),
                    )))?;
                    state
                        .workspaces
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                        })
                        .stacks
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.config = Loadable::Loading;
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Loadable::Loading,
                            outputs: Default::default(),
                            state: Default::default(),
                        });
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
