use std::collections::HashMap;

use ratatui_macros::{line, span};

use pulumi_automation::{
    event::{EngineEvent, EventType},
    stack::{Operation, ResourceState, ResourceType, StackChangeStep},
};
use ratatui::{
    style::{Color, Style},
    widgets::ListItem,
};

pub struct ResourceListItem {
    pub operation: Option<Operation>,
    pub resource_type: Option<ResourceType>,
    pub name: Option<String>,
    pub start: Option<i64>,
    pub end: Option<i64>,
    pub failed: bool,
}

impl<'a> Into<ListItem<'a>> for ResourceListItem {
    fn into(self) -> ListItem<'a> {
        let (op_style, op_text) = match self.operation {
            Some(op) => (
                Style::default().fg(match op {
                    Operation::Create => Color::LightGreen,
                    Operation::Update => Color::LightYellow,
                    Operation::Delete => Color::LightRed,
                    Operation::Same => Color::DarkGray,
                    _ => Color::LightBlue,
                }),
                op.to_string(),
            ),
            None => (Style::default(), String::new()),
        };

        let (start_style, start_text) = if let Some(start) = self.start {
            let end = self
                .end
                .unwrap_or_else(|| chrono::Utc::now().timestamp_millis() / 1000);
            let duration_s = end - start;
            (
                Style::default().fg(if self.end.is_some() {
                    Color::DarkGray
                } else {
                    Color::Gray
                }),
                format!("{:.3}s", duration_s),
            )
        } else {
            (Style::default(), String::new())
        };

        let (resource_type_style, resource_type_text) =
            if let Some(resource_type) = self.resource_type {
                (
                    Style::default().fg(match resource_type {
                        ResourceType::Stack => Color::DarkGray,
                        _ => Color::Gray,
                    }),
                    resource_type.to_string(),
                )
            } else {
                (Style::default(), String::new())
            };

        let name_text = self.name.clone().unwrap_or_default();

        ListItem::new(line![
            span!(op_style; "{:7}", op_text),
            span!(start_style; "{:<4}", start_text),
            span!(resource_type_style; "{:25}", resource_type_text),
            span!(Style::default(); "{:<}", name_text),
        ])
    }
}

impl<'a> From<&'a ResourceState> for ResourceListItem {
    fn from(resource_state: &'a ResourceState) -> Self {
        Self {
            resource_type: Some(resource_state.resource_type.clone()),
            name: Some(resource_state.name()),
            operation: None,
            start: None,
            end: None,
            failed: false,
        }
    }
}

impl<'a> From<&'a StackChangeStep> for ResourceListItem {
    fn from(step: &'a StackChangeStep) -> Self {
        if let Some(new_state) = &step.new_state {
            return Self {
                resource_type: Some(new_state.resource_type.clone()),
                name: Some(new_state.name()),
                operation: Some(step.op.clone()),
                start: None,
                end: None,
                failed: false,
            };
        } else if let Some(old_state) = &step.old_state {
            return Self {
                resource_type: Some(old_state.resource_type.clone()),
                name: Some(old_state.name()),
                operation: Some(step.op.clone()),
                start: None,
                end: None,
                failed: false,
            };
        } else {
            return Self {
                resource_type: None,
                name: None,
                operation: Some(step.op.clone()),
                start: None,
                end: None,
                failed: false,
            };
        }
    }
}

impl ResourceListItem {
    pub fn from_events<I>(events: I) -> Vec<Self>
    where
        I: IntoIterator<Item = EngineEvent>,
    {
        let mut list_items = HashMap::new();

        for event in events {
            match event.event {
                EventType::ResourcePreEvent { details, .. } => {
                    let item = ResourceListItem {
                        resource_type: Some(details.metadata.resource_type.clone()),
                        name: Some(details.metadata.name().clone()),
                        operation: Some(details.metadata.op.clone()),
                        start: event.timestamp,
                        end: None,
                        failed: false,
                    };

                    list_items.insert(details.metadata.urn.clone(), item);
                }
                EventType::ResOutputsEvent { details, .. } => {
                    list_items
                        .entry(details.metadata.urn.clone())
                        .and_modify(|item| {
                            item.end = event.timestamp;
                        })
                        .or_insert(ResourceListItem {
                            resource_type: Some(details.metadata.resource_type.clone()),
                            name: Some(details.metadata.name().clone()),
                            operation: Some(details.metadata.op.clone()),
                            start: event.timestamp,
                            end: None,
                            failed: false,
                        });
                }
                EventType::ResOpFailedEvent { details, .. } => {
                    list_items
                        .entry(details.metadata.urn.clone())
                        .and_modify(|item| {
                            item.end = event.timestamp;
                            item.failed = true;
                        })
                        .or_insert(ResourceListItem {
                            resource_type: Some(details.metadata.resource_type.clone()),
                            name: Some(details.metadata.name().clone()),
                            operation: Some(details.metadata.op.clone()),
                            start: event.timestamp,
                            end: None,
                            failed: true,
                        });
                }
                _ => continue,
            }
        }

        // Convert the HashMap into a Vec of ResourceListItem sorted by start time
        let mut items: Vec<ResourceListItem> = list_items.into_values().collect();
        items.sort_by(|a, b| {
            a.start
                .unwrap_or(0)
                .cmp(&b.start.unwrap_or(0))
                .then_with(|| a.name.cmp(&b.name))
        });

        items
    }
}
