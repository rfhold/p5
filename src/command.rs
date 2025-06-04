use crate::{AppState, actions::WorkspaceAction, state::Loadable};

pub fn parse_command_to_action(command: &str, state: &AppState) -> crate::Result<crate::AppAction> {
    let command = command.trim().to_lowercase();

    let parts = command.split_whitespace().collect::<Vec<&str>>();
    if parts.is_empty() {
        return Ok(crate::AppAction::ToastError("Esc to close".to_string()));
    }
    match parts[0] {
        "workspace" => {
            if parts.len() < 2 {
                return Ok(crate::AppAction::ToastError(
                    "Usage: workspace <path>".to_string(),
                ));
            }
            let path = parts[1].to_string();
            Ok(crate::AppAction::WorkspaceAction(
                WorkspaceAction::SelectWorkspace(path),
            ))
        }
        "stack" => {
            if parts.len() < 2 {
                return Ok(crate::AppAction::ToastError(
                    "Usage: stack <name>".to_string(),
                ));
            }
            let name = parts[1].to_string();

            if let Loadable::Loaded(workspace) = &state.workspace() {
                Ok(crate::AppAction::WorkspaceAction(
                    WorkspaceAction::SelectStack(workspace.clone(), name),
                ))
            } else {
                Ok(crate::AppAction::ToastError(
                    "Workspace not loaded".to_string(),
                ))
            }
        }
        command => Ok(crate::AppAction::ToastError(format!(
            "Unknown command: '{}'",
            command
        ))),
    }
}
