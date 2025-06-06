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
