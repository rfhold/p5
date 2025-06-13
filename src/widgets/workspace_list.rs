use ratatui::{
    layout::Alignment,
    widgets::{Block, List, ListItem, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppState, Loadable};

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
            .border_type(ratatui::widgets::BorderType::Rounded);

        match state.workspaces() {
            Loadable::Loaded(workspaces) => {
                let items = workspaces
                    .iter()
                    .map(|w| ListItem::new(w.cwd.clone()))
                    .collect::<Vec<_>>();

                let list = List::new(items).block(block);
                StatefulWidget::render(list, area, buf, &mut Default::default());
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
