use ratatui::{
    buffer::Buffer,
    layout::{Margin, Rect},
    style::Style,
    widgets::{
        Block, List, ListItem, ListState, Scrollbar, ScrollbarOrientation, ScrollbarState,
        StatefulWidget,
    },
};

#[derive(Default, Clone, Debug)]
pub struct ScrollableListState {
    pub list_state: ListState,
}

pub struct ScrollableList<'a> {
    items: Vec<ListItem<'a>>,
    block: Option<Block<'a>>,
}

impl<'a> ScrollableList<'a> {
    pub fn new(items: Vec<ListItem<'a>>) -> Self {
        Self { items, block: None }
    }

    pub fn block(mut self, block: Block<'a>) -> Self {
        self.block = Some(block);
        self
    }

    fn content_height(&self) -> usize {
        self.items.iter().map(|item| item.height()).sum::<usize>()
    }

    fn v_scroll_state(&self, list_state: &ListState, area: &Rect) -> Option<ScrollbarState> {
        let content_height = self.content_height();
        let area_height = area.height as usize;
        if content_height <= area_height {
            return None;
        }
        let position = self
            .items
            .iter()
            .take(list_state.selected().unwrap_or(0))
            .map(|item| item.height())
            .sum::<usize>();

        let overflow = content_height - area_height;
        let offset_position = position.saturating_sub(overflow);

        Some(ScrollbarState::new(overflow).position(offset_position))
    }

    fn v_scroll_area(&self, area: Rect) -> Rect {
        if let Some(_block) = &self.block {
            area.inner(Margin {
                horizontal: 0,
                vertical: 1,
            })
        } else {
            area
        }
    }
}

impl StatefulWidget for ScrollableList<'_> {
    type State = ScrollableListState;

    fn render(self, area: Rect, buf: &mut Buffer, state: &mut Self::State) {
        let v_scroll_area = self.v_scroll_area(area);
        let v_scroll_state = self.v_scroll_state(&state.list_state, &v_scroll_area);

        let list = List::new(self.items)
            .block(self.block.unwrap_or_default())
            .highlight_style(Style::default().fg(ratatui::style::Color::Yellow));

        StatefulWidget::render(list, area, buf, &mut state.list_state);

        if v_scroll_state.is_none() {
            return;
        }

        let v_scrollbar = Scrollbar::new(ScrollbarOrientation::VerticalRight);
        StatefulWidget::render(
            v_scrollbar,
            v_scroll_area,
            buf,
            &mut v_scroll_state.unwrap(),
        );
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use insta::assert_snapshot;
    use ratatui::{Terminal, backend::TestBackend};

    #[test]
    fn test_basic_scrollable() {
        let list_items = vec![
            ListItem::new("Item 1"),
            ListItem::new("Item 2"),
            ListItem::new("Item 3"),
            ListItem::new("Item 4"),
        ];

        let list = ScrollableList::new(list_items).block(
            Block::default()
                .title("Scrollable List")
                .borders(ratatui::widgets::Borders::ALL),
        );

        let mut state = ScrollableListState {
            list_state: ListState::default(),
        };

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(list, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }

    #[test]
    fn test_overflow() {
        let list_items = vec![
            ListItem::new("Item 1"),
            ListItem::new("Item 2"),
            ListItem::new("Item 3"),
            ListItem::new(
                "Item 4, which is a bit longer than the others to test overflow. Turns out, it still needs to be a bit longer",
            ),
        ];

        let list = ScrollableList::new(list_items).block(
            Block::default()
                .title("Scrollable List")
                .borders(ratatui::widgets::Borders::ALL),
        );

        let mut state = ScrollableListState {
            list_state: ListState::default(),
        };

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(list, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }

    #[test]
    fn test_vertical_scroll() {
        let list_items = (1..=35)
            .map(|i| ListItem::new(format!("Item {}", i)))
            .collect::<Vec<_>>();

        let list = ScrollableList::new(list_items).block(
            Block::default()
                .title("Scrollable List")
                .borders(ratatui::widgets::Borders::ALL),
        );

        let mut state = ScrollableListState {
            list_state: ListState::default().with_selected(Some(30)),
        };

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(list, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }

    #[test]
    fn test_vertical_scroll_first_page() {
        let list_items = (1..=35)
            .map(|i| ListItem::new(format!("Item {}", i)))
            .collect::<Vec<_>>();

        let list = ScrollableList::new(list_items).block(
            Block::default()
                .title("Scrollable List")
                .borders(ratatui::widgets::Borders::ALL),
        );

        let mut state = ScrollableListState {
            list_state: ListState::default().with_selected(Some(17)),
        };

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(list, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }

    #[test]
    fn test_vertical_scroll_last_page() {
        let list_items = (1..=35)
            .map(|i| ListItem::new(format!("Item {}", i)))
            .collect::<Vec<_>>();

        let list = ScrollableList::new(list_items).block(
            Block::default()
                .title("Scrollable List")
                .borders(ratatui::widgets::Borders::ALL),
        );

        let mut state = ScrollableListState {
            list_state: ListState::default().with_selected(Some(34)),
        };

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(list, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }
}
