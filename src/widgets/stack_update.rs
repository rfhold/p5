use ratatui::{
    layout::{Alignment, Direction, Layout},
    widgets::{Block, List, ListItem, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppState, Loadable};

use super::resource_list_item::ResourceListItem;

#[derive(Default, Clone)]
pub struct StackUpdate {}

impl StatefulWidget for StackUpdate {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let change_summary = state.stack_update_preview();
        let events = state.stack_update_events();

        let title = match change_summary {
            Loadable::Loaded(_) => match events {
                Loadable::Loaded(e) => {
                    if e.done {
                        "Update - Complete"
                    } else {
                        "Update - In Progress"
                    }
                }
                Loadable::Loading => "Update - Loading Events",
                Loadable::NotLoaded => "Update - Waiting for Events",
            },
            Loadable::Loading => "Update - Loading Summary",
            Loadable::NotLoaded => "Update - Idle",
        };

        let layout_constraints = if events.is_loaded() {
            vec![
                ratatui::layout::Constraint::Percentage(50),
                ratatui::layout::Constraint::Percentage(50),
            ]
        } else {
            vec![ratatui::layout::Constraint::Percentage(100)]
        };

        let layout = Layout::default()
            .direction(Direction::Horizontal)
            .constraints(layout_constraints)
            .split(area);

        let summary_area = if events.is_loaded() {
            layout[1]
        } else {
            layout[0]
        };

        if let Loadable::Loaded(events) = events {
            // TODO: How do I avoid the clone here?
            let items: Vec<ListItem> = ResourceListItem::from_events(events.events.clone())
                .into_iter()
                .map(|item| item.into())
                .collect();

            let block = Block::bordered()
                .title(title)
                .border_type(ratatui::widgets::BorderType::Rounded);

            Widget::render(List::new(items).block(block.clone()), layout[0], buf);
        }

        let block = Block::bordered()
            .title(if events.is_loaded() {
                "Preview - Summary".to_string()
            } else {
                "Preview".to_string()
            })
            .border_type(ratatui::widgets::BorderType::Rounded);

        match change_summary {
            Loadable::Loaded(change_summary) => {
                let items: Vec<ListItem> = change_summary
                    .steps
                    .iter()
                    .map(|step| ResourceListItem::from(step).into())
                    .collect();

                Widget::render(List::new(items).block(block), summary_area, buf);
            }
            Loadable::Loading => {
                Paragraph::new("Loading...".to_string())
                    .block(block)
                    .alignment(Alignment::Left)
                    .render(summary_area, buf);
            }
            Loadable::NotLoaded => {
                Paragraph::new("No Stack Selected".to_string())
                    .block(block)
                    .alignment(Alignment::Left)
                    .render(summary_area, buf);
            }
        };
    }
}
