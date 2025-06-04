use ratatui::{
    layout::{Constraint, Flex, Layout, Rect},
    style::{Color, Style},
    widgets::{Clear, Paragraph, StatefulWidget, Widget},
};

use crate::{AppContext, AppState, widgets::OperationView};

#[derive(Clone, Default)]
pub struct AppLayout {}

impl StatefulWidget for AppLayout {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        OperationView::default().render(area, buf, state);

        if let AppContext::CommandPrompt = state.current_context() {
            let popup_area = popup_area(area, 40, 3, Flex::Center, Flex::Center);
            let command_prompt = Paragraph::new(state.command_prompt.value())
                .block(
                    ratatui::widgets::Block::bordered()
                        .title("Command")
                        .border_type(ratatui::widgets::BorderType::Rounded),
                )
                .alignment(ratatui::layout::Alignment::Left);

            Clear.render(popup_area, buf);
            command_prompt.render(popup_area, buf);
        }

        if let Some((expr, toast)) = state.toast.as_ref() {
            if chrono::Utc::now() > *expr {
                state.toast = None;
            } else {
                let popup_area = popup_area(area, 40, 3, Flex::End, Flex::End);
                let toast_message = Paragraph::new(toast.clone())
                    .block(
                        ratatui::widgets::Block::bordered()
                            .title("Attention")
                            .border_style(Style::default().fg(Color::Yellow))
                            .border_type(ratatui::widgets::BorderType::Rounded),
                    )
                    .alignment(ratatui::layout::Alignment::Left);

                Clear.render(popup_area, buf);
                toast_message.render(popup_area, buf);
            }
        }
    }
}

fn popup_area(area: Rect, length_x: u16, length_y: u16, v_flex: Flex, h_flex: Flex) -> Rect {
    let vertical = Layout::vertical([Constraint::Length(length_y)]).flex(v_flex);
    let horizontal = Layout::horizontal([Constraint::Length(length_x)]).flex(h_flex);
    let [area] = vertical.areas(area);
    let [area] = horizontal.areas(area);
    area
}
