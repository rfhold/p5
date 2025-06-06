use pulumi_automation::stack::{Operation, ResourceState, ResourceType, StackChangeStep};
use ratatui::{
    style::{Color, Style},
    text::{Line, Span},
    widgets::ListItem,
};

pub struct ResourceListItem<'resource> {
    pub operation: Option<Operation>,
    pub resource_state: Option<&'resource ResourceState>,
}

impl<'a> Into<ListItem<'a>> for ResourceListItem<'a> {
    fn into(self) -> ListItem<'a> {
        if let Some(resource) = self.resource_state {
            let mut line = Line::default();

            if let Some(op) = self.operation {
                line.push_span(Span::styled(
                    op.to_string(),
                    Style::default().fg(match op {
                        Operation::Create => Color::LightGreen,
                        Operation::Update => Color::LightYellow,
                        Operation::Delete => Color::LightRed,
                        Operation::Same => Color::DarkGray,
                        _ => Color::LightBlue,
                    }),
                ));

                line.push_span(" ");
            }

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
        } else {
            ListItem::new(Line::default())
        }
    }
}

impl<'a> From<&'a ResourceState> for ResourceListItem<'a> {
    fn from(resource_state: &'a ResourceState) -> Self {
        Self {
            resource_state: Some(resource_state),
            operation: None,
        }
    }
}

impl<'a> From<&'a StackChangeStep> for ResourceListItem<'a> {
    fn from(step: &'a StackChangeStep) -> Self {
        if step.new_state.is_none() {
            return Self {
                resource_state: step.old_state.as_ref(),
                operation: Some(step.op.clone()),
            };
        }
        Self {
            resource_state: step.new_state.as_ref(),
            operation: Some(step.op.clone()),
        }
    }
}
