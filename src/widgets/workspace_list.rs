use ratatui::{
    layout::Alignment,
    style::Style,
    widgets::{Block, List, ListItem, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppContext, AppState, Loadable};

use super::theme::color;

pub struct WorkspaceList {}

impl WorkspaceList {
    pub fn new() -> Self {
        WorkspaceList {}
    }
}

impl StatefulWidget for WorkspaceList {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let block = Block::bordered()
            .title("Workspaces")
            .border_type(ratatui::widgets::BorderType::Rounded)
            .border_style(Style::default().fg(
                if AppContext::WorkspaceList == state.background_context() {
                    color::BORDER_HIGHLIGHT
                } else {
                    color::BORDER_DEFAULT
                },
            ));

        match state.workspaces() {
            Loadable::Loaded(workspaces) => {
                let items = workspaces
                    .iter()
                    .map(|w| ListItem::new(w.cwd.clone()))
                    .collect::<Vec<_>>();

                let list = List::new(items)
                    .block(block)
                    .highlight_style(Style::default().fg(color::SELECTED));
                StatefulWidget::render(list, area, buf, &mut state.workspace_list_state);
            }
            Loadable::Loading => {
                Paragraph::new("Loading Workspaces...".to_string())
                    .block(block)
                    .alignment(Alignment::Left)
                    .render(area, buf);
            }
            Loadable::NotLoaded => {
                Paragraph::new("Not Loaded".to_string())
                    .block(block)
                    .alignment(Alignment::Left)
                    .render(area, buf);
            }
        };
    }
}
