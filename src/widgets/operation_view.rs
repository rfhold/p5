use ratatui::{
    buffer::Buffer,
    layout::{Alignment, Rect},
    widgets::{Block, Paragraph, StatefulWidget, Widget},
};

use crate::{AppState, state::Loadable};

#[derive(Clone, Default)]
pub struct OperationView {}

impl StatefulWidget for OperationView {
    type State = AppState;

    fn render(self, area: Rect, buf: &mut Buffer, state: &mut Self::State) {
        let workspace = state.workspace();

        let (title, message) = match workspace {
            Loadable::Loaded(workspace) => match state.stack() {
                Loadable::Loaded(stack) => {
                    let title = format!("Workspace: {} - Stack: {}", workspace.cwd, stack.name);

                    match state.stack_outputs() {
                        Loadable::Loaded(outputs) => (
                            title,
                            serde_json::to_string_pretty(&outputs)
                                .unwrap_or_else(|_| "Failed to serialize outputs".to_string()),
                        ),
                        Loadable::Loading => (title, "Loading Outputs...".to_string()),
                        Loadable::NotLoaded => (title, "No Outputs Available".to_string()),
                    }
                }
                Loadable::Loading => (
                    format!("Workspace: {}", workspace.cwd),
                    "Loading Stack...".to_string(),
                ),
                Loadable::NotLoaded => (
                    format!("Workspace: {}", workspace.cwd),
                    "No Stack Selected".to_string(),
                ),
            },
            Loadable::Loading => (
                "Loading Workspace...".to_string(),
                "Please wait...".to_string(),
            ),
            Loadable::NotLoaded => (
                "No Workspace Loaded".to_string(),
                "Please load a workspace.".to_string(),
            ),
        };
        let block = Block::bordered()
            .title(title)
            .border_type(ratatui::widgets::BorderType::Rounded);

        let paragraph = Paragraph::new(message)
            .block(block)
            .alignment(Alignment::Left);

        paragraph.render(area, buf);
    }
}
