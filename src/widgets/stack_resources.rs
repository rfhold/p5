use ratatui::{
    layout::Alignment,
    widgets::{Block, List, ListItem, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppState, Loadable};

use super::resource_list_item::ResourceListItem;

#[derive(Default, Clone)]
pub struct StackResources {}

impl StatefulWidget for StackResources {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let block = Block::bordered()
            .title("Resources")
            .border_type(ratatui::widgets::BorderType::Rounded);

        match state.stack_state_data() {
            Loadable::Loaded(state) => {
                let items: Vec<ListItem> = state
                    .deployment
                    .resources
                    .iter()
                    .map(|resource| ResourceListItem::new(resource).into())
                    .collect();

                Widget::render(List::new(items).block(block), area, buf);
            }
            Loadable::Loading => {
                Paragraph::new("Loading State...".to_string())
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
