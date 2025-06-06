use ratatui::{
    layout::Alignment,
    widgets::{Block, List, ListItem, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppState, Loadable};

use super::resource_list_item::ResourceListItem;

#[derive(Default, Clone)]
pub struct StackPreview {}

impl StatefulWidget for StackPreview {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let block = Block::bordered()
            .title("Preview")
            .border_type(ratatui::widgets::BorderType::Rounded);

        match state.stack_preview() {
            Loadable::Loaded(change_summary) => {
                let items: Vec<ListItem> = change_summary
                    .steps
                    .iter()
                    .map(|step| ResourceListItem::from(step).into())
                    .collect();

                Widget::render(List::new(items).block(block), area, buf);
            }
            Loadable::Loading => {
                Paragraph::new("Loading Preview...".to_string())
                    .block(block)
                    .alignment(Alignment::Left)
                    .render(area, buf);
            }
            Loadable::NotLoaded => {
                Paragraph::new("No Stack Selected".to_string())
                    .block(block)
                    .alignment(Alignment::Left)
                    .render(area, buf);
            }
        };
    }
}
