use crate::{
    AppState,
    actions::{StackAction, WorkspaceAction},
    state::{Loadable, OperationOptions, ProgramOperation, StackContext},
};

pub fn parse_command_to_action(
    command: &str,
    state: &AppState,
) -> crate::Result<Option<crate::AppAction>> {
    let command = command.trim().to_lowercase();

    let parts = command.split_whitespace().collect::<Vec<&str>>();
    if parts.is_empty() {
        return Ok(None);
    }
    match parts[0] {
        "workspaces" => Ok(Some(crate::AppAction::PushContext(
            crate::AppContext::WorkspaceList,
        ))),
        "workspace" => {
            if parts.len() < 2 {
                return Ok(Some(crate::AppAction::ToastError(
                    "Usage: workspace <path>".to_string(),
                )));
            }
            let path = parts[1].to_string();

            Ok(Some(crate::AppAction::WorkspaceAction(
                WorkspaceAction::SelectWorkspace(path),
            )))
        }
        "stack" => {
            if parts.len() < 2 {
                return Ok(Some(crate::AppAction::ToastError(
                    "Usage: stack <name>".to_string(),
                )));
            }
            let name = parts[1].to_string();

            if let Loadable::Loaded(workspace) = &state.workspace() {
                Ok(Some(crate::AppAction::WorkspaceAction(
                    WorkspaceAction::SelectStack(workspace.clone(), name),
                )))
            } else {
                Ok(Some(crate::AppAction::ToastError(
                    "No workspace selected".to_string(),
                )))
            }
        }
        "outputs" => Ok(Some(crate::AppAction::PushContext(
            crate::AppContext::Stack(StackContext::Outputs),
        ))),
        "config" => Ok(Some(crate::AppAction::PushContext(
            crate::AppContext::Stack(StackContext::Config),
        ))),
        "resources" => Ok(Some(crate::AppAction::PushContext(
            crate::AppContext::Stack(StackContext::Resources),
        ))),
        "preview" => {
            if let Loadable::Loaded(stack) = &state.stack() {
                Ok(Some(crate::AppAction::StackAction(
                    StackAction::RunProgram(
                        ProgramOperation::Update,
                        stack.clone(),
                        OperationOptions::default().preview_only(),
                    ),
                )))
            } else {
                Ok(Some(crate::AppAction::ToastError(
                    "No stack selected".to_string(),
                )))
            }
        }
        "update" => {
            if let Loadable::Loaded(stack) = &state.stack() {
                Ok(Some(crate::AppAction::StackAction(
                    StackAction::RunProgram(
                        ProgramOperation::Update,
                        stack.clone(),
                        OperationOptions::default(),
                    ),
                )))
            } else {
                Ok(Some(crate::AppAction::ToastError(
                    "No stack selected".to_string(),
                )))
            }
        }
        "refresh" => {
            if let Loadable::Loaded(stack) = &state.stack() {
                Ok(Some(crate::AppAction::StackAction(
                    StackAction::RunProgram(
                        ProgramOperation::Refresh,
                        stack.clone(),
                        OperationOptions::default(),
                    ),
                )))
            } else {
                Ok(Some(crate::AppAction::ToastError(
                    "No stack selected".to_string(),
                )))
            }
        }
        "destroy" => {
            if let Loadable::Loaded(stack) = &state.stack() {
                Ok(Some(crate::AppAction::StackAction(
                    StackAction::RunProgram(
                        ProgramOperation::Destroy,
                        stack.clone(),
                        OperationOptions::default(),
                    ),
                )))
            } else {
                Ok(Some(crate::AppAction::ToastError(
                    "No stack selected".to_string(),
                )))
            }
        }
        command => Ok(Some(crate::AppAction::ToastError(format!(
            "Unknown command: '{}'",
            command
        )))),
    }
}
