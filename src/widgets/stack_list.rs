use ratatui::{
    layout::Alignment,
    style::Style,
    widgets::{Block, List, ListItem, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppContext, AppState, Loadable};

use super::theme::color;

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
            .border_type(ratatui::widgets::BorderType::Rounded)
            .border_style(Style::default().fg(
                if AppContext::StackList == state.background_context() {
                    color::BORDER_HIGHLIGHT
                } else {
                    color::BORDER_DEFAULT
                },
            ));

        match state.stacks() {
            Loadable::Loaded(stacks) => {
                let items = stacks
                    .iter()
                    .map(|s| ListItem::new(s.name.clone()))
                    .collect::<Vec<_>>();

                let list = List::new(items)
                    .block(block)
                    .highlight_style(Style::default().fg(color::SELECTED));
                StatefulWidget::render(list, area, buf, &mut state.stack_list_state);
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
