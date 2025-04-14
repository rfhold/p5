use ratatui::{
    buffer::Buffer,
    layout::{self, Layout, Rect},
    style::{Color, Style},
    text::{Line, Span, Text},
    widgets::{BorderType, ListItem, Paragraph, StatefulWidget, Widget},
};

use crate::{
    model::StepViewModel,
    pulumi::{self, Operation},
};

use super::{MultiSelectState, json::json_value_to_lines, panel::SelectionPanel};

#[derive(Debug, Clone, Default)]
pub struct PreviewView {
    steps: Option<Vec<StepViewModel>>,
    error: Option<pulumi::CommandError>,
    active: bool,
}

impl PreviewView {
    pub fn new(steps: Option<Vec<StepViewModel>>, error: Option<pulumi::CommandError>) -> Self {
        PreviewView {
            steps,
            error,
            ..Default::default()
        }
    }

    pub fn with_steps(mut self, steps: Option<Vec<StepViewModel>>) -> Self {
        self.steps = steps;
        self
    }

    pub fn with_error(mut self, error: Option<pulumi::CommandError>) -> Self {
        self.error = error;
        self
    }

    pub fn with_active(mut self, active: bool) -> Self {
        self.active = active;
        self
    }
}

impl StatefulWidget for PreviewView {
    type State = MultiSelectState;

    fn render(self, area: Rect, buf: &mut Buffer, state: &mut Self::State) {
        let layout = Layout::default()
            .direction(layout::Direction::Horizontal)
            .constraints(
                [
                    layout::Constraint::Percentage(50),
                    layout::Constraint::Percentage(50),
                ]
                .as_ref(),
            )
            .split(area);

        let left_area = layout[0];
        let right_area = layout[1];

        if let Some(err) = self.error {
            let stdout = err.stdout;
            let paragraph = Paragraph::new(stdout)
                .block(
                    ratatui::widgets::Block::default()
                        .title("Error")
                        .borders(ratatui::widgets::Borders::ALL),
                )
                .style(Style::default().fg(ratatui::style::Color::Red));

            paragraph.render(area, buf);
            return;
        }

        let panel = SelectionPanel::new(self.steps.as_deref())
            .with_title("Preview".to_string())
            .with_highlighted(Some(state.selected.clone()))
            .with_highlighted_style(Some(Style::default().bg(Color::DarkGray)))
            .with_active(self.active);

        panel.render(left_area, buf, &mut state.list_state);

        if let Some(i) = state.list_state.selected() {
            if let Some(steps) = self.steps.as_ref() {
                if let Some(step) = steps.get(i) {
                    let paragraph: Paragraph = step.clone().into();
                    paragraph
                        .block(
                            ratatui::widgets::Block::default()
                                .title(format!("Step {}", i + 1))
                                .border_type(BorderType::Rounded)
                                .borders(ratatui::widgets::Borders::ALL),
                        )
                        .render(right_area, buf);
                }
            }
        }
    }
}

impl<'a> Into<ListItem<'a>> for StepViewModel {
    fn into(self) -> ListItem<'a> {
        let item = Text::from(Line::from(vec![
            Span::from(format!("{}", self.step.op)).style(Style::default().fg(self.step.op.into())),
            Span::from(format!(": {}", self.step.urn)),
        ]));

        return ListItem::new(item);
    }
}

impl<'a> Into<Paragraph<'a>> for StepViewModel {
    fn into(self) -> Paragraph<'a> {
        let mut item = Text::default();

        match self.step.op {
            Operation::Create | Operation::Import | Operation::CreateReplacement => {
                if let Some(new_state) = self.step.new_state {
                    if let Some(inputs) = new_state.inputs {
                        item.extend(
                            json_value_to_lines(inputs)
                                .iter()
                                .map(|line| line.clone().style(Style::default().fg(Color::Green))),
                        );
                    }
                    if let Some(outputs) = new_state.outputs {
                        item.extend(json_value_to_lines(outputs));
                    }
                }
            }
            Operation::Update => {
                if let Some(old_state) = self.step.old_state {
                    if let Some(outputs) = old_state.outputs {
                        item.extend(
                            json_value_to_lines(outputs)
                                .iter()
                                .map(|line| line.clone().style(Style::default().fg(Color::Yellow))),
                        );
                    }
                }
                if let Some(new_state) = self.step.new_state {
                    if let Some(inputs) = new_state.inputs {
                        item.extend(
                            json_value_to_lines(inputs)
                                .iter()
                                .map(|line| line.clone().style(Style::default().fg(Color::Green))),
                        );
                    }
                    if let Some(outputs) = new_state.outputs {
                        item.extend(json_value_to_lines(outputs));
                    }
                }
            }
            Operation::Delete | Operation::DeleteReplaced => {
                if let Some(old_state) = self.step.old_state {
                    if let Some(outputs) = old_state.outputs {
                        item.extend(
                            json_value_to_lines(outputs)
                                .iter()
                                .map(|line| line.clone().style(Style::default().fg(Color::Red))),
                        );
                    }
                }
            }
            _ => {}
        }

        Paragraph::new(item)
    }
}
impl Into<Color> for Operation {
    fn into(self) -> Color {
        match self {
            Operation::Same => Color::Gray,
            Operation::Read => Color::Blue,
            Operation::Create | Operation::Import | Operation::CreateReplacement => Color::Green,
            Operation::Update => Color::Yellow,
            Operation::Delete | Operation::DeleteReplaced => Color::Red,
            Operation::Replace => Color::Magenta,
        }
    }
}
