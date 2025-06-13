use p5::controller::Action;
use pulumi_automation::{
    event::EngineEvent,
    local::{LocalStack, LocalWorkspace},
    stack::StackChangeSummary,
    workspace::{Deployment, OutputMap, StackSettings, StackSummary},
};
use tokio::sync::mpsc;

use crate::{
    AppContext, AppState,
    state::{
        Loadable, OperationContext, OperationEvents, OperationOptions, OperationProgress,
        ProgramOperation, StackContext, StackOutputs, StackState, WorkspaceState,
    },
    tasks::{AppTask, stack::StackTask, workspace::WorkspaceTask},
};

#[derive(Clone)]
pub enum AppAction {
    Exit,
    SubmitCommand(String),
    ToastError(String),
    PopContext,
    PushContext(AppContext),
    WorkspaceAction(WorkspaceAction),
    StackAction(StackAction),
    ListWorkspaces,
    PersistWorkspaces(Vec<LocalWorkspace>),
}

#[derive(Clone)]
pub enum WorkspaceAction {
    SelectWorkspace(String),
    PersistWorkspace(LocalWorkspace),
    SelectStack(LocalWorkspace, String),
    ListStacks(LocalWorkspace),
    PersistStacks(LocalWorkspace, Vec<StackSummary>),
    PersistStack(LocalWorkspace, LocalStack),
    PersistStackOutputs(LocalWorkspace, LocalStack, OutputMap),
    PersistStackConfig(LocalWorkspace, LocalStack, StackSettings),
    PersistStackState(LocalWorkspace, LocalStack, Deployment),
    LoadStackState(LocalWorkspace, LocalStack),
    LoadStackOutputs(LocalWorkspace, LocalStack),
    LoadStackConfig(LocalWorkspace, LocalStack),
}

#[derive(Clone)]
pub enum StackAction {
    RunProgram(ProgramOperation, LocalStack, OperationOptions),
    BeginOperation(ProgramOperation, LocalStack, OperationOptions),
    PersistChangeSummary(ProgramOperation, LocalStack, StackChangeSummary),
    PersistEvent(ProgramOperation, LocalStack, EngineEvent),
    PersistOperationDone(ProgramOperation, LocalStack),
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
                                crate::state::StackContext::Operation(_) => {}
                            }
                        }
                    }
                    AppContext::WorkspaceList => {
                        action_tx.try_send(AppAction::ListWorkspaces)?;
                    }
                    _ => {}
                }

                Ok(())
            }
            AppAction::PopContext => {
                if !state.context_stack.is_empty() {
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
            AppAction::ListWorkspaces => {
                state.workspaces = Loadable::Loading;
                task_tx.try_send(AppTask::ListWorkspaces)?;
                Ok(())
            }
            AppAction::PersistWorkspaces(workspaces) => {
                state.workspaces = Loadable::Loaded(workspaces.clone());
                Ok(())
            }
            AppAction::WorkspaceAction(action) => match action {
                WorkspaceAction::ListStacks(workspace) => {
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Loadable::Loading,
                            stack_store: Default::default(),
                        });
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::ListStacks(
                        workspace.clone(),
                    )))?;
                    Ok(())
                }
                WorkspaceAction::PersistStacks(workspace, stacks) => {
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .and_modify(|w| {
                            w.stacks = Loadable::Loaded(stacks.clone());
                        })
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Loadable::Loaded(stacks.clone()),
                            stack_store: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::SelectWorkspace(cwd) => {
                    state.selected_workspace = Some(WorkspaceState {
                        workspace_path: cwd.clone(),
                        selected_stack: None,
                    });
                    state.workspace_store.entry(cwd.clone()).or_insert_with(|| {
                        crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loading,
                            stacks: Loadable::Loading,
                            stack_store: Default::default(),
                        }
                    });
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::SelectWorkspace(
                        cwd.clone(),
                    )))?;
                    action_tx.try_send(AppAction::PushContext(AppContext::StackList))?;
                    Ok(())
                }
                WorkspaceAction::PersistWorkspace(workspace) => {
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .and_modify(|w| {
                            if let Loadable::Loading = w.workspace {
                                *w = crate::state::WorkspaceOutputs {
                                    workspace: Loadable::Loaded(workspace.clone()),
                                    stacks: Loadable::Loading,
                                    stack_store: Default::default(),
                                };
                            }
                        })
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Loadable::Loading,
                            stack_store: Default::default(),
                        });

                    action_tx.try_send(AppAction::WorkspaceAction(WorkspaceAction::ListStacks(
                        workspace.clone(),
                    )))?;

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
                        resource_state: Default::default(),
                    });
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(name.clone())
                        .and_modify(|s| {
                            s.stack = Loadable::Loading;
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loading,
                            config: Loadable::Loading,
                            outputs: Default::default(),
                            state: Default::default(),
                            operation: Default::default(),
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
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.stack = Loadable::Loaded(stack.clone());
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Loadable::Loading,
                            outputs: Default::default(),
                            state: Default::default(),
                            operation: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::PersistStackOutputs(workspace, stack, outputs) => {
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.outputs = Loadable::Loaded(outputs.clone());
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            outputs: Loadable::Loaded(outputs.clone()),
                            config: Default::default(),
                            state: Default::default(),
                            operation: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::PersistStackConfig(workspace, stack, config) => {
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.config = Loadable::Loaded(config.clone());
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Loadable::Loaded(config.clone()),
                            outputs: Default::default(),
                            state: Default::default(),
                            operation: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::PersistStackState(workspace, stack, stack_state) => {
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.state = Loadable::Loaded(stack_state.clone());
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Default::default(),
                            outputs: Default::default(),
                            state: Loadable::Loaded(stack_state.clone()),
                            operation: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::LoadStackState(workspace, stack) => {
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::GetStackState(
                        workspace.clone(),
                        stack.clone(),
                    )))?;
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.state = Loadable::Loading;
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Default::default(),
                            outputs: Default::default(),
                            state: Loadable::Loading,
                            operation: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::LoadStackOutputs(workspace, stack) => {
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::GetStackOutputs(
                        workspace.clone(),
                        stack.clone(),
                    )))?;
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.outputs = Loadable::Loading;
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Default::default(),
                            outputs: Loadable::Loading,
                            state: Default::default(),
                            operation: Default::default(),
                        });
                    Ok(())
                }
                WorkspaceAction::LoadStackConfig(workspace, stack) => {
                    task_tx.try_send(AppTask::WorkspaceTask(WorkspaceTask::GetStackConfig(
                        workspace.clone(),
                        stack.clone(),
                    )))?;
                    state
                        .workspace_store
                        .entry(workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(stack.name.clone())
                        .and_modify(|s| {
                            s.config = Loadable::Loading;
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(stack.clone()),
                            config: Loadable::Loading,
                            outputs: Default::default(),
                            state: Default::default(),
                            operation: Default::default(),
                        });
                    Ok(())
                }
            },
            AppAction::StackAction(action) => match action {
                StackAction::RunProgram(operation, local_stack, options) => {
                    action_tx.try_send(AppAction::PushContext(AppContext::Stack(
                        StackContext::Operation(OperationContext::Summary),
                    )))?;
                    action_tx.try_send(AppAction::StackAction(StackAction::BeginOperation(
                        operation.clone(),
                        local_stack.clone(),
                        options.clone(),
                    )))?;

                    Ok(())
                }
                StackAction::BeginOperation(operation, local_stack, options) => {
                    task_tx.try_send(AppTask::StackTask(StackTask::RunOperation(
                        operation.clone(),
                        local_stack.clone(),
                        options.clone(),
                    )))?;
                    state
                        .workspace_store
                        .entry(local_stack.workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(local_stack.workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(local_stack.name.clone())
                        .and_modify(|s| {
                            s.operation = Some(OperationProgress {
                                operation: operation.clone(),
                                options: Some(options.clone()),
                                change_summary: Loadable::Loading,
                                events: Default::default(),
                            });
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(local_stack.clone()),
                            config: Default::default(),
                            outputs: Default::default(),
                            state: Default::default(),
                            operation: Some(OperationProgress {
                                operation: operation.clone(),
                                options: Some(options.clone()),
                                change_summary: Loadable::Loading,
                                events: Default::default(),
                            }),
                        });
                    Ok(())
                }
                StackAction::PersistChangeSummary(operation, local_stack, stack_change_summary) => {
                    state
                        .workspace_store
                        .entry(local_stack.workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(local_stack.workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(local_stack.name.clone())
                        .and_modify(|s| {
                            if let Some(op) = &mut s.operation {
                                op.change_summary = Loadable::Loaded(stack_change_summary.clone());
                            } else {
                                s.operation = Some(OperationProgress {
                                    operation: operation.clone(),
                                    change_summary: Loadable::Loaded(stack_change_summary.clone()),
                                    events: Default::default(),
                                    options: Default::default(),
                                });
                            }
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(local_stack.clone()),
                            config: Default::default(),
                            outputs: Default::default(),
                            state: Default::default(),
                            operation: Some(OperationProgress {
                                operation: operation.clone(),
                                change_summary: Loadable::Loaded(stack_change_summary.clone()),
                                events: Default::default(),
                                options: Default::default(),
                            }),
                        });
                    Ok(())
                }
                StackAction::PersistEvent(operation, local_stack, engine_event) => {
                    let outputs = state
                        .workspace_store
                        .entry(local_stack.workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(local_stack.workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(local_stack.name.clone())
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(local_stack.clone()),
                            config: Default::default(),
                            outputs: Default::default(),
                            state: Default::default(),
                            operation: Some(OperationProgress {
                                operation: operation.clone(),
                                change_summary: Loadable::default(),
                                events: Loadable::Loaded(OperationEvents {
                                    events: vec![engine_event.clone()],
                                    states: vec![],
                                    done: false,
                                }),
                                options: Default::default(),
                            }),
                        });

                    let operation = outputs
                        .operation
                        .as_mut()
                        .expect("Operation should be present");

                    let events = operation.events.as_mut_or_default(OperationEvents {
                        events: vec![],
                        states: vec![],
                        done: false,
                    });

                    if let Err(err) = events.apply_event(engine_event.clone()) {
                        tracing::error!("Failed to apply engine event: {}", err);
                    }

                    Ok(())
                }
                StackAction::PersistOperationDone(op, local_stack) => {
                    state
                        .workspace_store
                        .entry(local_stack.workspace.cwd.clone())
                        .or_insert_with(|| crate::state::WorkspaceOutputs {
                            workspace: Loadable::Loaded(local_stack.workspace.clone()),
                            stacks: Default::default(),
                            stack_store: Default::default(),
                        })
                        .stack_store
                        .entry(local_stack.name.clone())
                        .and_modify(|s| {
                            match &mut s.operation {
                                Some(op) => {
                                    let events = op.events.as_mut_or_default(OperationEvents {
                                        events: vec![],
                                        states: vec![],
                                        done: true,
                                    });
                                    events.done = true;
                                    op.events = Loadable::Loaded(events.clone());
                                    op
                                }
                                None => {
                                    s.operation = Some(OperationProgress {
                                        operation: op.clone(),
                                        change_summary: Loadable::default(),
                                        events: Loadable::Loaded(OperationEvents {
                                            events: vec![],
                                            states: vec![],
                                            done: true,
                                        }),
                                        options: Default::default(),
                                    });
                                    s.operation.as_mut().unwrap()
                                }
                            };
                        })
                        .or_insert_with(|| StackOutputs {
                            stack: Loadable::Loaded(local_stack.clone()),
                            config: Default::default(),
                            outputs: Default::default(),
                            state: Default::default(),
                            operation: Some(OperationProgress {
                                operation: op.clone(),
                                change_summary: Loadable::default(),
                                events: Loadable::Loaded(OperationEvents {
                                    events: vec![],
                                    states: vec![],
                                    done: true,
                                }),
                                options: Default::default(),
                            }),
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
