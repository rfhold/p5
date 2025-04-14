use ratatui::{
    buffer::Buffer,
    layout::Rect,
    widgets::{BorderType, Paragraph, StatefulWidget, Widget},
};

use crate::model::StackView;

use super::{json::JsonView, preview::PreviewView};

#[derive(Default, Debug, Clone)]
pub struct View {
    active: bool,
}

impl View {
    pub fn with_active(mut self, active: bool) -> Self {
        self.active = active;
        self
    }
}

impl StatefulWidget for View {
    type State = StackView;

    fn render(self, area: Rect, buf: &mut Buffer, state: &mut Self::State) {
        match state {
            StackView::Output { output, .. } => {
                JsonView::new(output.clone()).render(area, buf);
            }
            StackView::Preview {
                steps,
                command_error,
                state,
            } => {
                PreviewView::new(steps.clone(), command_error.clone())
                    .with_active(self.active)
                    .render(area, buf, state);
            }
            StackView::None => Paragraph::new("Nothing Selected")
                .block(
                    ratatui::widgets::Block::default()
                        .borders(ratatui::widgets::Borders::ALL)
                        .border_type(BorderType::Rounded),
                )
                .render(area, buf),
        };
    }
}
