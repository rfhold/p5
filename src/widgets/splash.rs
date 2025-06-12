use ratatui::widgets::Widget;

pub struct SplashScreen {
    pub title: String,
    pub message: String,
}

impl SplashScreen {
    pub fn new(title: String, message: String) -> Self {
        Self { title, message }
    }
}

impl Widget for SplashScreen {
    fn render(self, area: ratatui::prelude::Rect, buf: &mut ratatui::prelude::Buffer) {
        let title_block = ratatui::widgets::Block::bordered()
            .title(self.title)
            .border_type(ratatui::widgets::BorderType::Rounded);

        let message_paragraph = ratatui::widgets::Paragraph::new(self.message)
            .block(title_block)
            .alignment(ratatui::layout::Alignment::Center);

        message_paragraph.render(area, buf);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use insta::assert_snapshot;
    use ratatui::{Terminal, backend::TestBackend};

    #[test]
    fn test_render_splash_screen() {
        let splash_screen = SplashScreen::new(
            "P5".to_string(),
            "Press ':' to open the command prompt. Ctrl+C to exit.".to_string(),
        );

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_widget(splash_screen, frame.area()))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }
}
