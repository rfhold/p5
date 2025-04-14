use ratatui::{
    buffer::Buffer,
    layout::Rect,
    style::Style,
    text::{Line, Span},
    widgets::{Paragraph, Widget},
};

pub struct JsonView {
    pub json: Option<serde_json::Value>,
}

impl JsonView {
    pub fn new(json: Option<serde_json::Value>) -> Self {
        JsonView { json }
    }
}

impl Widget for JsonView {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let json_str = match self.json {
            Some(json) => {
                serde_json::to_string_pretty(&json).unwrap_or_else(|_| "Invalid JSON".to_string())
            }
            None => "No JSON data".to_string(),
        };

        let paragraph = Paragraph::new(json_str)
            .block(
                ratatui::widgets::Block::default()
                    .title("JSON")
                    .borders(ratatui::widgets::Borders::ALL),
            )
            .style(Style::default());

        paragraph.render(area, buf);
    }
}

pub fn json_value_to_lines(json: serde_json::Value) -> Vec<Line<'static>> {
    match serde_json::to_string_pretty(&json) {
        Ok(json_str) => json_str
            .lines()
            .map(|line| Line::from(Span::from(line.to_string())))
            .collect(),
        Err(_) => vec![Line::from(Span::raw("Invalid JSON"))],
    }
}
