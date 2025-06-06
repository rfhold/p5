use pulumi_automation::stack::{ResourceState, ResourceType};
use ratatui::{
    style::{Color, Style},
    text::{Line, Span},
    widgets::ListItem,
};

pub struct ResourceListItem<'resource> {
    pub resource_state: &'resource ResourceState,
}

impl<'resource> ResourceListItem<'resource> {
    pub fn new(resource_state: &'resource ResourceState) -> Self {
        Self { resource_state }
    }
}

impl<'a> Into<ListItem<'a>> for ResourceListItem<'a> {
    fn into(self) -> ListItem<'a> {
        let resource = self.resource_state;

        let mut line = Line::default();

        line.push_span(Span::styled(
            resource.resource_type.to_string(),
            Style::default().fg(match resource.resource_type {
                ResourceType::Stack => Color::DarkGray,
                _ => Color::Gray,
            }),
        ));

        line.push_span(" ");

        line.push_span(resource.name().to_string());

        ListItem::new(line)
    }
}
