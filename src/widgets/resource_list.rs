use pulumi_automation::stack::{Operation, ResourceState, StackChangeSummary};
use ratatui::{
    buffer::Buffer,
    layout::Rect,
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, ListItem, StatefulWidget},
};
use ratatui_macros::{line, span};

use super::scrollable_list::{ScrollableList, ScrollableListState};
use crate::state::ResourceOperationState;

#[derive(Debug, Clone)]
pub struct ResourceItem<'a> {
    pub urn: String,
    pub line: Line<'a>,
}

pub trait ResourceAdapter {
    fn get_items(&self) -> Vec<ResourceItem>;
}

#[derive(Default, Clone, Debug)]
pub struct ResourceListState {
    pub scrollable_state: ScrollableListState,
    pub excluded_resources: Vec<String>,
    pub target_resources: Vec<String>,
    pub replace_resources: Vec<String>,
}

pub struct ResourceList<'a, A: ResourceAdapter> {
    pub block: Block<'a>,
    pub adapter: A,
}

// Adapter implementations
pub struct CurrentStateAdapter<'a>(pub &'a Vec<ResourceState>);

impl<'a> ResourceAdapter for CurrentStateAdapter<'a> {
    fn get_items(&self) -> Vec<ResourceItem> {
        self.0
            .iter()
            .map(|resource| ResourceItem {
                urn: resource.urn.clone(),
                line: line!(
                    span!(Style::default().fg(Color::White); "{:25} ", resource.resource_type),
                    span!(Style::default().fg(Color::Cyan); "{:<}", resource.name()),
                ),
            })
            .collect()
    }
}

pub struct ChangeSummaryAdapter<'a>(pub &'a StackChangeSummary);

impl<'a> ResourceAdapter for ChangeSummaryAdapter<'a> {
    fn get_items(&self) -> Vec<ResourceItem> {
        self.0
            .steps
            .iter()
            .map(|step| {
                let state = if let Some(s) = step.new_state.as_ref() {
                    Some(s)
                } else {
                    step.old_state.as_ref()
                };

                if let Some(ref state) = state {
                    ResourceItem {
                        urn: state.urn.clone(),
                        line: line!(
                            operation_to_span(&step.op, Some(10)),
                            span!(Style::default().fg(Color::Gray); "{:<25} ", state.resource_type),
                            span!(Style::default(); "{:<}", state.name()),
                        ),
                    }
                } else {
                    ResourceItem {
                        urn: String::new(),
                        line: line!(
                            operation_to_span(&step.op, Some(10)),
                            span!(Style::default().fg(Color::Gray); "{:<25} ", "Unknown resource type"),
                            span!(Style::default(); "{:<}", "Unknown resource"),
                        ),
                    }
                }
            })
            .collect()
    }
}

pub struct OperationStateAdapter<'a>(pub &'a Vec<ResourceOperationState>);

impl<'a> ResourceAdapter for OperationStateAdapter<'a> {
    fn get_items(&self) -> Vec<ResourceItem> {
        self.0
            .iter()
            .map(|op_state| {
                let metadata = match op_state {
                    ResourceOperationState::InProgress { pre_event, .. } => &pre_event.metadata,
                    ResourceOperationState::Completed { out_event, .. } => &out_event.metadata,
                    ResourceOperationState::Failed { failed_event, .. } => &failed_event.metadata,
                };

                let pre_event = match op_state {
                    ResourceOperationState::InProgress { pre_event, .. } => pre_event,
                    ResourceOperationState::Completed { pre_event, .. } => pre_event,
                    ResourceOperationState::Failed { pre_event, .. } => pre_event,
                };

                let (dur_text, dur_style) = {
                    let (start, end) = match op_state {
                        ResourceOperationState::InProgress { start_time, .. } => {
                            (*start_time, chrono::Utc::now())
                        }
                        ResourceOperationState::Completed {
                            start_time,
                            end_time,
                            ..
                        } => (*start_time, *end_time),
                        ResourceOperationState::Failed {
                            start_time,
                            end_time,
                            ..
                        } => (*start_time, *end_time),
                    };

                    let duration_s = (end - start).num_seconds();
                    (
                        format!("{:.0}s", duration_s as f64),
                        Style::default().fg(match op_state {
                            ResourceOperationState::InProgress { .. } => Color::Gray,
                            ResourceOperationState::Completed { .. } => Color::DarkGray,
                            ResourceOperationState::Failed { .. } => Color::Red,
                        }),
                    )
                };

                let resource_type = metadata.resource_type.to_string();

                let name = metadata.name();

                ResourceItem {
                    urn: metadata.urn.clone(),
                    line: line!(
                        operation_to_span(&pre_event.metadata.op, Some(10)),
                        span!(dur_style; "{:<2} ", dur_text),
                        span!(Style::default().fg(Color::Gray); "{:.25} ", resource_type),
                        span!(Style::default(); "{:<}", name),
                    ),
                }
            })
            .collect()
    }
}

impl<'a, A: ResourceAdapter> StatefulWidget for ResourceList<'a, A> {
    type State = ResourceListState;

    fn render(self, area: Rect, buf: &mut Buffer, state: &mut Self::State) {
        let items = self.adapter.get_items();

        let list_items: Vec<ListItem> = items
            .iter()
            .map(|item| {
                let mut style = Style::default();

                // Apply selection styling (priority order matters)
                if state.excluded_resources.contains(&item.urn) {
                    style = style.fg(Color::Red).add_modifier(Modifier::CROSSED_OUT);
                } else if state.replace_resources.contains(&item.urn) {
                    style = style.fg(Color::Yellow).add_modifier(Modifier::ITALIC);
                } else if state.target_resources.contains(&item.urn) {
                    style = style.fg(Color::Green).add_modifier(Modifier::BOLD);
                }

                ListItem::new(item.line.clone()).style(style)
            })
            .collect();

        let scrollable_list = ScrollableList::new(list_items).block(self.block);
        scrollable_list.render(area, buf, &mut state.scrollable_state);
    }
}

impl<'a> ResourceList<'a, CurrentStateAdapter<'a>> {
    pub fn from_states(block: Block<'a>, states: &'a Vec<ResourceState>) -> Self {
        ResourceList {
            block,
            adapter: CurrentStateAdapter(states),
        }
    }
}

impl<'a> ResourceList<'a, ChangeSummaryAdapter<'a>> {
    pub fn from_summary(block: Block<'a>, summary: &'a StackChangeSummary) -> Self {
        ResourceList {
            block,
            adapter: ChangeSummaryAdapter(summary),
        }
    }
}

impl<'a> ResourceList<'a, OperationStateAdapter<'a>> {
    pub fn from_operations(block: Block<'a>, operations: &'a Vec<ResourceOperationState>) -> Self {
        ResourceList {
            block,
            adapter: OperationStateAdapter(operations),
        }
    }
}

fn operation_to_span(op: &Operation, width: Option<usize>) -> Span<'_> {
    let style = match op {
        Operation::Create => Style::default().fg(Color::Green),
        Operation::Update => Style::default().fg(Color::Yellow),
        Operation::Delete => Style::default().fg(Color::Red),
        Operation::Read => Style::default().fg(Color::Blue),
        Operation::Replace => Style::default().fg(Color::Magenta),
        _ => Style::default(),
    };

    match width {
        Some(w) => span!(style; "{:<w$}", op.to_string()),
        None => span!(style; "{}", op.to_string()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use insta::assert_snapshot;
    use pulumi_automation::{
        event::{ResOpFailedDetails, ResOutputsDetails, ResourcePreDetails},
        stack::{ResourceType, StackChangeStep},
    };
    use ratatui::{Terminal, backend::TestBackend, widgets::ListState};

    #[test]
    fn test_resources() {
        let items = vec![
            ResourceState {
                urn: "urn:pulumi:stack::project::pulumi:pulumi:Stack::project".to_string(),
                resource_type: ResourceType::Stack,
                ..Default::default()
            },
            ResourceState {
                urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::my-bucket".to_string(),
                resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                ..Default::default()
            },
        ];

        let mut state = ResourceListState {
            scrollable_state: ScrollableListState {
                list_state: ListState::default(),
            },
            excluded_resources: vec![],
            target_resources: vec![],
            replace_resources: vec![],
        };

        let widget = ResourceList::from_states(
            Block::default()
                .title("Resources")
                .borders(ratatui::widgets::Borders::ALL),
            &items,
        );

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(widget, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }

    #[test]
    fn test_change_summary() {
        let change_summary = StackChangeSummary {
            steps: vec![
                StackChangeStep {
                    op: pulumi_automation::stack::Operation::Create,
                    old_state: None,
                    new_state: Some(ResourceState {
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::my-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        ..Default::default()
                    }),
                    ..Default::default()
                },
                StackChangeStep {
                    op: pulumi_automation::stack::Operation::Delete,
                    old_state: Some(ResourceState {
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::old-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        ..Default::default()
                    }),
                    new_state: None,
                    ..Default::default()
                },
            ],
            ..Default::default()
        };

        let mut state = ResourceListState {
            scrollable_state: ScrollableListState {
                list_state: ListState::default(),
            },
            excluded_resources: vec![],
            target_resources: vec![],
            replace_resources: vec![],
        };

        let widget = ResourceList::from_summary(
            Block::default()
                .title("Change Summary")
                .borders(ratatui::widgets::Borders::ALL),
            &change_summary,
        );

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(widget, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }

    #[test]
    fn test_operation_states() {
        let operation_states = vec![
            ResourceOperationState::InProgress {
                sequence: 1,
                start_time: chrono::Utc::now(),
                pre_event: ResourcePreDetails {
                    metadata: pulumi_automation::event::StepEventMetadata {
                        op: pulumi_automation::stack::Operation::Create,
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::my-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        old: None,
                        new: None,
                        keys: None,
                        diffs: None,
                        detailed_diff: None,
                        logical: None,
                        provider: "aws".to_string(),
                    },
                    planning: Some(false),
                    extra_values: None,
                },
            },
            ResourceOperationState::Completed {
                sequence: 2,
                start_time: chrono::Utc::now(),
                end_time: chrono::Utc::now(),
                pre_event: ResourcePreDetails {
                    metadata: pulumi_automation::event::StepEventMetadata {
                        op: pulumi_automation::stack::Operation::Update,
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::my-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        old: None,
                        new: None,
                        keys: None,
                        diffs: None,
                        detailed_diff: None,
                        logical: None,
                        provider: "aws".to_string(),
                    },
                    planning: Some(false),
                    extra_values: None,
                },
                out_event: ResOutputsDetails {
                    metadata: pulumi_automation::event::StepEventMetadata {
                        op: pulumi_automation::stack::Operation::Update,
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::my-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        old: None,
                        new: None,
                        keys: None,
                        diffs: None,
                        detailed_diff: None,
                        logical: None,
                        provider: "aws".to_string(),
                    },
                    planning: Some(false),
                    extra_values: None,
                },
            },
            ResourceOperationState::Failed {
                sequence: 3,
                start_time: chrono::Utc::now(),
                end_time: chrono::Utc::now(),
                pre_event: ResourcePreDetails {
                    metadata: pulumi_automation::event::StepEventMetadata {
                        op: pulumi_automation::stack::Operation::Delete,
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::my-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        old: None,
                        new: None,
                        keys: None,
                        diffs: None,
                        detailed_diff: None,
                        logical: None,
                        provider: "aws".to_string(),
                    },
                    planning: Some(false),
                    extra_values: None,
                },
                failed_event: ResOpFailedDetails {
                    metadata: pulumi_automation::event::StepEventMetadata {
                        op: pulumi_automation::stack::Operation::Delete,
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::my-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        old: None,
                        new: None,
                        keys: None,
                        diffs: None,
                        detailed_diff: None,
                        logical: None,
                        provider: "aws".to_string(),
                    },
                    steps: 0,
                    status: 0,
                    extra_values: None,
                },
            },
        ];

        let mut state = ResourceListState {
            scrollable_state: ScrollableListState {
                list_state: ListState::default(),
            },
            excluded_resources: vec![],
            target_resources: vec![],
            replace_resources: vec![],
        };

        let widget = ResourceList::from_operations(
            Block::default()
                .title("Resource Operations")
                .borders(ratatui::widgets::Borders::ALL),
            &operation_states,
        );

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(widget, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }
}
