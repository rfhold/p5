use crate::{
    actions::WorkspaceAction,
    state::{Loadable, StackContext},
    AppState,
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
        command => Ok(Some(crate::AppAction::ToastError(format!(
            "Unknown command: '{}'",
            command
        )))),
    }
}
