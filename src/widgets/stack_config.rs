use ratatui::{
    layout::Alignment,
    widgets::{Block, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppState, Loadable};

#[derive(Default, Clone)]
pub struct StackConfig {}

impl StatefulWidget for StackConfig {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let config = match state.stack_config() {
            Loadable::Loaded(config) => Some(
                serde_json::to_string_pretty(&config)
                    .unwrap_or_else(|_| "Failed to serialize config".to_string()),
            ),
            Loadable::Loading => Some("Loading Config...".to_string()),
            Loadable::NotLoaded => None,
        };

        if let Some(config) = config {
            let config_block = Block::bordered()
                .title("Config")
                .border_type(ratatui::widgets::BorderType::Rounded);

            let config_paragraph = Paragraph::new(config)
                .block(config_block)
                .alignment(Alignment::Left);

            config_paragraph.render(area, buf);
        }
    }
}
