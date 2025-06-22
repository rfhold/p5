use ratatui::{
    layout::{Constraint, Direction, Flex, Layout, Rect},
    style::{Color, Style},
    widgets::{Clear, Paragraph, StatefulWidget, Widget},
};

use crate::{
    AppContext, AppState,
    widgets::{StackLayout, StackList, WorkspaceList},
};

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
        let current_context = state.current_context();
        let background_context = state.background_context();
        match &background_context {
            AppContext::CommandPrompt => {}
            _ => {
                let main_layout = Layout::default()
                    .direction(Direction::Horizontal)
                    .constraints([Constraint::Percentage(25), Constraint::Percentage(75)])
                    .split(area);

                let sidebar_area = main_layout[0];
                let main_area = main_layout[1];

                let sidebar_layout = Layout::default()
                    .direction(Direction::Vertical)
                    .constraints([Constraint::Percentage(50), Constraint::Percentage(50)])
                    .split(sidebar_area);

                WorkspaceList::new().render(sidebar_layout[0], buf, state);
                StackList::new().render(sidebar_layout[1], buf, state);

                StackLayout::new(state.stack_context()).render(main_area, buf, state);
            }
        }

        if let AppContext::CommandPrompt = current_context {
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::state::tests::create_test_app_state_with_fixtures;
    use insta::assert_snapshot;
    use ratatui::{Terminal, backend::TestBackend};

    #[test]
    fn test_layout_complete_update() {
        let mut state = create_test_app_state_with_fixtures(true);
        let mut terminal = Terminal::new(TestBackend::new(120, 40)).unwrap();

        terminal
            .draw(|frame| {
                let layout = AppLayout::default();
                frame.render_stateful_widget(layout, frame.area(), &mut state);
            })
            .unwrap();

        assert_snapshot!(terminal.backend())
    }

    #[test]
    fn test_layout_in_progress_update() {
        let mut state = create_test_app_state_with_fixtures(false);
        let mut terminal = Terminal::new(TestBackend::new(120, 40)).unwrap();

        terminal
            .draw(|frame| {
                let layout = AppLayout::default();
                frame.render_stateful_widget(layout, frame.area(), &mut state);
            })
            .unwrap();

        assert_snapshot!(terminal.backend())
    }
}
