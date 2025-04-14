use std::fmt::Display;

use ratatui::prelude::*;
use ratatui::{
    text::Line,
    widgets::{Block, BorderType, Borders, List, ListItem, ListState, StatefulWidget},
};

#[derive(Debug, Clone, Default)]
pub struct SelectionPanel<'a, T: Into<ListItem<'a>>> {
    items: Option<&'a [T]>,
    title: Option<String>,
    highlighted: Option<Vec<usize>>,
    highlighted_style: Option<Style>,
    active: bool,
}

impl<'a, T: Into<ListItem<'a>>> SelectionPanel<'a, T> {
    pub fn new(items: Option<&'a [T]>) -> Self {
        SelectionPanel {
            items,
            title: None,
            active: false,
            highlighted: None,
            highlighted_style: None,
        }
    }

    pub fn with_items(mut self, items: Option<&'a [T]>) -> Self {
        self.items = items;
        self
    }

    pub fn with_highlighted(mut self, highlighted: Option<Vec<usize>>) -> Self {
        self.highlighted = highlighted;
        self
    }

    pub fn with_highlighted_style(mut self, highlighted_style: Option<Style>) -> Self {
        self.highlighted_style = highlighted_style;
        self
    }

    pub fn with_title(mut self, title: String) -> Self {
        self.title = Some(title);
        self
    }

    pub fn with_active(mut self, active: bool) -> Self {
        self.active = active;
        self
    }

    fn num_items(&self) -> usize {
        if let Some(items) = self.items {
            items.len()
        } else {
            0
        }
    }
}

impl<'a, T: Into<ListItem<'a>>> StatefulWidget for SelectionPanel<'a, T>
where
    T: Clone,
{
    type State = ListState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let num_items = self.num_items();

        let bottom_title = match num_items {
            0 => "No items".to_string(),
            _ => format!("{} of {}", state.selected().unwrap_or(0) + 1, num_items),
        };
        let mut block = Block::default()
            .border_type(BorderType::Rounded)
            .borders(Borders::ALL)
            .title_bottom(Line::from(bottom_title).right_aligned())
            .border_style(Style::default().fg(if self.active {
                Color::Green
            } else {
                Color::White
            }));

        if let Some(title) = &self.title {
            block = block.title(title.as_str());
        }

        let items = self.items.unwrap_or(&[]);

        let highlighted_style = self.highlighted_style.unwrap_or(Style::default());

        let list = List::new(items.iter().enumerate().map(|(i, item)| {
            item.clone()
                .into()
                .style(if let Some(highlighted) = &self.highlighted {
                    if highlighted.contains(&i) {
                        highlighted_style
                    } else {
                        Style::default()
                    }
                } else {
                    Style::default()
                })
        }))
        .block(block)
        .highlight_style(match self.active {
            true => Style::default().bg(Color::Blue).fg(Color::Black),
            false => Style::default().bg(Color::Gray).fg(Color::Black),
        });

        StatefulWidget::render(&list, area, buf, state);
    }
}
