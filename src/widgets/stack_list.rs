use ratatui::{
    layout::Alignment,
    widgets::{Block, List, ListItem, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppState, Loadable};

pub struct StackList {}

impl StackList {
    pub fn new() -> Self {
        StackList {}
    }
}

impl StatefulWidget for StackList {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let block = Block::bordered()
            .title("Stacks")
            .border_type(ratatui::widgets::BorderType::Rounded);

        match state.stacks() {
            Loadable::Loaded(stacks) => {
                let items = stacks
                    .iter()
                    .map(|s| ListItem::new(s.name.clone()))
                    .collect::<Vec<_>>();

                let list = List::new(items).block(block);
                StatefulWidget::render(list, area, buf, &mut Default::default());
            }
            Loadable::Loading => {
                Paragraph::new("Loading Stacks...".to_string())
                    .block(block)
                    .alignment(Alignment::Left)
                    .render(area, buf);
            }
            Loadable::NotLoaded => {
                Paragraph::new("No Workspace Selected".to_string())
                    .block(block)
                    .alignment(Alignment::Left)
                    .render(area, buf);
            }
        };
    }
}
