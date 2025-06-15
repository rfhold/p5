use ratatui::{
    layout::Alignment,
    style::Style,
    widgets::{Block, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppContext, AppState, Loadable, StackContext};

use super::{resource_list::ResourceList, theme::color};

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
            .border_type(ratatui::widgets::BorderType::Rounded)
            .border_style(Style::default().fg(
                if let AppContext::Stack(StackContext::Resources) = state.background_context() {
                    color::BORDER_HIGHLIGHT
                } else {
                    color::BORDER_DEFAULT
                },
            ));

        if let Some((data, selection)) = state.stack_resource_state() {
            match data {
                Loadable::Loaded(stack_state) => {
                    ResourceList::from_states(block, &stack_state.deployment.resources)
                        .render(area, buf, selection);
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
        } else {
            Paragraph::new("No Stack Selected".to_string())
                .block(block)
                .alignment(Alignment::Left)
                .render(area, buf);
        }
    }
}
