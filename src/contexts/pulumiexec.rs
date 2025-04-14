use std::sync::Arc;

use p5::{AppEvent, ContextAction, EventType};
use tokio::sync::{RwLock, mpsc};

use crate::{
    model::{Model, ResourceList, ResourceModel, StackList, StackModel, StackView, StepViewModel},
    pulumi::{self, Interactor, StackOutputOption},
    ui::MultiSelectState,
};

use super::{Action, PulumiAction};

#[derive(Debug, Clone, Default, PartialEq, Eq, Hash)]
pub struct PulumiActionHandler {}

impl PulumiActionHandler {
    pub async fn handle_action(
        &self,
        state: Arc<RwLock<Model>>,
        action: Action,
        action_signal: mpsc::Sender<ContextAction<Action>>,
    ) {
        match action {
            Action::PulumiAction(pulumi_action) => match pulumi_action {
                PulumiAction::Output(program, stack) => {
                    match program.stack_output(&stack, vec![StackOutputOption::ShowSecrets(true)]) {
                        Ok(output) => {
                            action_signal
                                .send(ContextAction::AppAction(Action::PulumiAction(
                                    PulumiAction::StoreOutput(Some(output), None),
                                )))
                                .await
                                .expect("Failed to send output");
                        }
                        Err(err) => match err {
                            pulumi::PulumiExecutionError::CommandError(e) => {
                                action_signal
                                    .send(ContextAction::AppAction(Action::PulumiAction(
                                        PulumiAction::StoreOutput(None, Some(e)),
                                    )))
                                    .await
                                    .expect("Failed to send command error");
                            }
                            _ => {
                                action_signal
                                    .send(ContextAction::AppAction(Action::Event(AppEvent {
                                        message: format!("Error getting stack output: {:?}", err),
                                        timestamp: chrono::Utc::now().to_string(),
                                        event_type: EventType::Error,
                                        source: "PulumiTask".to_string(),
                                        command_result: None,
                                    })))
                                    .await
                                    .expect("Failed to send event");
                                return;
                            }
                        },
                    }
                }
                PulumiAction::StoreOutput(value, command_error) => {
                    let mut state = state.write().await;
                    state.set_selected_stack_output(value, command_error);
                }
                PulumiAction::Preview(program, stack, options) => {
                    match program.stack_preview(&stack, options.unwrap_or(vec![])) {
                        Ok(preview) => {
                            action_signal
                                .send(ContextAction::AppAction(Action::PulumiAction(
                                    PulumiAction::StorePreview(Some(preview), None),
                                )))
                                .await
                                .expect("Failed to send preview");

                            return;
                        }
                        Err(err) => match err {
                            pulumi::PulumiExecutionError::CommandError(e) => {
                                action_signal
                                    .send(ContextAction::AppAction(Action::PulumiAction(
                                        PulumiAction::StorePreview(None, Some(e)),
                                    )))
                                    .await
                                    .expect("Failed to send command error");
                            }
                            _ => {
                                action_signal
                                    .send(ContextAction::AppAction(Action::Event(AppEvent {
                                        message: format!("Error previewing stack: {:?}", err),
                                        timestamp: chrono::Utc::now().to_string(),
                                        event_type: EventType::Error,
                                        source: "PulumiTask".to_string(),
                                        command_result: None,
                                    })))
                                    .await
                                    .expect("Failed to send event");
                                return;
                            }
                        },
                    }
                }
                PulumiAction::StorePreview(stack_preview, command_error) => {
                    let mut state = state.write().await;
                    state.set_selected_stack_view(Some(StackView::Preview {
                        steps: if let Some(preview) = stack_preview {
                            Some(
                                preview
                                    .steps
                                    .into_iter()
                                    .map(|s| StepViewModel {
                                        step: s,
                                        ..Default::default()
                                    })
                                    .collect(),
                            )
                        } else {
                            None
                        },
                        command_error,
                        state: MultiSelectState::default(),
                    }));
                }
                PulumiAction::StackList(program) => {
                    let stacks = program.stack_list(vec![]);
                    match stacks {
                        Ok(stacks) => {
                            action_signal
                                .send(ContextAction::AppAction(Action::PulumiAction(
                                    PulumiAction::StoreStackList(Some(stacks)),
                                )))
                                .await
                                .expect("Failed to send stack list");
                        }
                        Err(err) => {
                            action_signal
                                .send(ContextAction::AppAction(Action::Event(AppEvent {
                                    message: format!("Error listing stacks: {:?}", err),
                                    timestamp: chrono::Utc::now().to_string(),
                                    event_type: EventType::Error,
                                    source: "PulumiTask".to_string(),
                                    command_result: None,
                                })))
                                .await
                                .expect("Failed to send error event");
                        }
                    }
                }
                PulumiAction::StoreStackList(stacks) => {
                    let mut state = state.write().await;
                    state.set_stack_list(match stacks {
                        Some(stacks) => Some(StackList {
                            stacks: stacks
                                .into_iter()
                                .map(|s| StackModel {
                                    stack: s,
                                    ..Default::default()
                                })
                                .collect(),
                            ..Default::default()
                        }),
                        None => None,
                    });
                }
                PulumiAction::ListStackResources(program, stack) => {
                    match program.stack_export(&stack, vec![]) {
                        Ok(export) => {
                            action_signal
                                .send(ContextAction::AppAction(Action::PulumiAction(
                                    PulumiAction::StoreStackResources(export.deployment.resources),
                                )))
                                .await
                                .expect("Failed to send stack resources");
                        }
                        Err(error) => {
                            action_signal
                                .send(ContextAction::AppAction(Action::Event(AppEvent {
                                    message: format!("Error listing resources: {:?}", error),
                                    timestamp: chrono::Utc::now().to_string(),
                                    event_type: EventType::Error,
                                    source: "PulumiTask".to_string(),
                                    command_result: None,
                                })))
                                .await
                                .expect("Failed to send error event");
                        }
                    }
                }
                PulumiAction::StoreStackResources(resources) => {
                    let mut state = state.write().await;
                    state.set_resource_list(match resources {
                        Some(resources) => Some(ResourceList {
                            resources: resources
                                .into_iter()
                                .map(|r| ResourceModel { resource: r })
                                .collect(),
                            ..Default::default()
                        }),
                        None => None,
                    });
                }
            },
            _ => todo!(),
        }
    }
}
