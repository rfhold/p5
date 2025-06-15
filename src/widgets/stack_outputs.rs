use ratatui::{
    layout::Alignment,
    style::Style,
    widgets::{Block, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppContext, AppState, Loadable, StackContext};

use super::theme::color;

#[derive(Default, Clone)]
pub struct StackOutputs {}

impl StatefulWidget for StackOutputs {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let outputs = match state.stack_outputs() {
            Loadable::Loaded(outputs) => Some(
                serde_json::to_string_pretty(&outputs)
                    .unwrap_or_else(|_| "Failed to serialize outputs".to_string()),
            ),
            Loadable::Loading => Some("Loading Outputs...".to_string()),
            Loadable::NotLoaded => None,
        };

        if let Some(outputs) = outputs {
            let outputs_block = Block::bordered()
                .title("Outputs")
                .border_type(ratatui::widgets::BorderType::Rounded)
                .border_style(Style::default().fg(
                    if let AppContext::Stack(StackContext::Outputs) = state.background_context() {
                        color::BORDER_HIGHLIGHT
                    } else {
                        color::BORDER_DEFAULT
                    },
                ));

            let outputs_paragraph = Paragraph::new(outputs)
                .block(outputs_block)
                .alignment(Alignment::Left);

            outputs_paragraph.render(area, buf);
        }
    }
}
