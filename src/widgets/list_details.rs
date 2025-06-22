use pulumi_automation::stack::{ResourceState, StackChangeSummary};
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
pub struct DetailItem<'a> {
    pub label: String,
    pub line: Line<'a>,
}

pub trait DetailAdapter {
    fn get_details(&self, selected_index: Option<usize>) -> Vec<DetailItem>;
    fn get_title(&self, selected_index: Option<usize>) -> String;
}

#[derive(Default, Clone, Debug)]
pub struct ListDetailsState {
    pub scrollable_state: ScrollableListState,
    pub selected_index: Option<usize>,
}

pub struct ListDetails<'a, A: DetailAdapter> {
    pub block: Block<'a>,
    pub adapter: A,
}

// Adapter implementations
pub struct CurrentStateDetailAdapter<'a>(&'a Vec<ResourceState>);

impl DetailAdapter for CurrentStateDetailAdapter<'_> {
    fn get_details(&self, selected_index: Option<usize>) -> Vec<DetailItem> {
        let Some(index) = selected_index else {
            return vec![DetailItem {
                label: "info".to_string(),
                line: line![span!["No resource selected"]],
            }];
        };

        let Some(resource) = self.0.get(index) else {
            return vec![DetailItem {
                label: "error".to_string(),
                line: line![span!["Invalid selection"]],
            }];
        };

        let mut details = vec![
            DetailItem {
                label: "URN".to_string(),
                line: line![span![resource.urn.clone()]],
            },
            DetailItem {
                label: "Type".to_string(),
                line: line![span![format!("{:?}", resource.resource_type)]],
            },
        ];

        if let Some(ref inputs) = resource.inputs {
            if inputs.is_object() {
                if let Some(obj) = inputs.as_object() {
                    details.push(DetailItem {
                        label: "Inputs".to_string(),
                        line: line![span![format!("{} properties", obj.len())]],
                    });
                }
            }
        }

        if let Some(ref outputs) = resource.outputs {
            if outputs.is_object() {
                if let Some(obj) = outputs.as_object() {
                    details.push(DetailItem {
                        label: "Outputs".to_string(),
                        line: line![span![format!("{} properties", obj.len())]],
                    });
                }
            }
        }

        if let Some(ref id) = resource.id {
            details.push(DetailItem {
                label: "ID".to_string(),
                line: line![span![id.clone()]],
            });
        }

        if let Some(ref provider) = resource.provider {
            details.push(DetailItem {
                label: "Provider".to_string(),
                line: line![span![provider.clone()]],
            });
        }

        details
    }

    fn get_title(&self, selected_index: Option<usize>) -> String {
        match selected_index.and_then(|i| self.0.get(i)) {
            Some(resource) => {
                let short_urn = resource.urn.split("::").last().unwrap_or(&resource.urn);
                format!("Resource Details: {}", short_urn)
            }
            None => "Resource Details".to_string(),
        }
    }
}

pub struct ChangeSummaryDetailAdapter<'a>(&'a StackChangeSummary);

impl DetailAdapter for ChangeSummaryDetailAdapter<'_> {
    fn get_details(&self, selected_index: Option<usize>) -> Vec<DetailItem> {
        let Some(index) = selected_index else {
            return vec![DetailItem {
                label: "info".to_string(),
                line: line![span!["No change selected"]],
            }];
        };

        let Some(step) = self.0.steps.get(index) else {
            return vec![DetailItem {
                label: "error".to_string(),
                line: line![span!["Invalid selection"]],
            }];
        };

        vec![
            DetailItem {
                label: "Operation".to_string(),
                line: line![operation_to_span(format!("{:?}", step.op))],
            },
            DetailItem {
                label: "URN".to_string(),
                line: line![span![step.urn.clone()]],
            },
            DetailItem {
                label: "Type".to_string(),
                line: line![span![match &step.new_state {
                    Some(state) => format!("{:?}", state.resource_type),
                    None => match &step.old_state {
                        Some(state) => format!("{:?}", state.resource_type),
                        None => "Unknown".to_string(),
                    },
                }]],
            },
        ]
    }

    fn get_title(&self, _selected_index: Option<usize>) -> String {
        "Change Diff".to_string()
    }
}

pub struct OperationDetailAdapter<'a>(&'a Vec<ResourceOperationState>);

impl DetailAdapter for OperationDetailAdapter<'_> {
    fn get_details(&self, selected_index: Option<usize>) -> Vec<DetailItem> {
        let Some(index) = selected_index else {
            return vec![DetailItem {
                label: "info".to_string(),
                line: line![span!["No operation selected"]],
            }];
        };

        let Some(operation_state) = self.0.get(index) else {
            return vec![DetailItem {
                label: "error".to_string(),
                line: line![span!["Invalid selection"]],
            }];
        };

        let metadata = match operation_state {
            ResourceOperationState::InProgress { pre_event, .. } => &pre_event.metadata,
            ResourceOperationState::Completed { out_event, .. } => &out_event.metadata,
            ResourceOperationState::Failed { failed_event, .. } => &failed_event.metadata,
        };

        let mut details = vec![
            DetailItem {
                label: "URN".to_string(),
                line: line![span![metadata.urn.clone()]],
            },
            DetailItem {
                label: "Operation".to_string(),
                line: line![operation_to_span(format!("{:?}", metadata.op))],
            },
            DetailItem {
                label: "Status".to_string(),
                line: line![span![match operation_state {
                    ResourceOperationState::InProgress { .. } => "In Progress",
                    ResourceOperationState::Completed { .. } => "Completed",
                    ResourceOperationState::Failed { .. } => "Failed",
                }]],
            },
            DetailItem {
                label: "Resource Type".to_string(),
                line: line![span![format!("{:?}", metadata.resource_type)]],
            },
        ];

        if let Some(ref old) = metadata.old {
            details.push(DetailItem {
                label: "Old Inputs".to_string(),
                line: line![span![format!("{} properties", old.inputs.len())]],
            });
        }

        if let Some(ref new) = metadata.new {
            details.push(DetailItem {
                label: "New Inputs".to_string(),
                line: line![span![format!("{} properties", new.inputs.len())]],
            });
        }

        details
    }

    fn get_title(&self, _selected_index: Option<usize>) -> String {
        "Operation Details".to_string()
    }
}

impl<A: DetailAdapter> StatefulWidget for ListDetails<'_, A> {
    type State = ListDetailsState;

    fn render(self, area: Rect, buf: &mut Buffer, state: &mut Self::State) {
        let details = self.adapter.get_details(state.selected_index);
        let title = self.adapter.get_title(state.selected_index);

        let list_items: Vec<ListItem> = details
            .iter()
            .map(|detail| {
                let mut formatted_line = line![
                    span![detail.label.clone()].style(
                        Style::default()
                            .fg(Color::Cyan)
                            .add_modifier(Modifier::BOLD)
                    ),
                    span![": "]
                ];

                // Append the spans from the detail line
                formatted_line.spans.extend(detail.line.spans.clone());

                ListItem::new(formatted_line)
            })
            .collect();

        let block = self.block.title(title);
        let scrollable_list = ScrollableList::new(list_items).block(block);
        scrollable_list.render(area, buf, &mut state.scrollable_state);
    }
}

impl<'a> ListDetails<'a, CurrentStateDetailAdapter<'a>> {
    pub fn from_states(block: Block<'a>, states: &'a Vec<ResourceState>) -> Self {
        ListDetails {
            block,
            adapter: CurrentStateDetailAdapter(states),
        }
    }
}

impl<'a> ListDetails<'a, ChangeSummaryDetailAdapter<'a>> {
    pub fn from_summary(block: Block<'a>, summary: &'a StackChangeSummary) -> Self {
        ListDetails {
            block,
            adapter: ChangeSummaryDetailAdapter(summary),
        }
    }
}

impl<'a> ListDetails<'a, OperationDetailAdapter<'a>> {
    pub fn from_operations(block: Block<'a>, operations: &'a Vec<ResourceOperationState>) -> Self {
        ListDetails {
            block,
            adapter: OperationDetailAdapter(operations),
        }
    }
}

fn operation_to_span(operation: String) -> Span<'static> {
    match operation.as_str() {
        "Create" => span![operation].style(Style::default().fg(Color::Green)),
        "Update" => span![operation].style(Style::default().fg(Color::Yellow)),
        "Delete" => span![operation].style(Style::default().fg(Color::Red)),
        "Replace" => span![operation].style(Style::default().fg(Color::Magenta)),
        _ => span![operation],
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use insta::assert_snapshot;
    use pulumi_automation::{
        event::{ResourcePreDetails, StepEventMetadata},
        stack::{Operation, ResourceType},
    };
    use ratatui::{Terminal, backend::TestBackend, widgets::ListState};

    #[test]
    fn test_current_state_details() {
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

        let mut state = ListDetailsState {
            scrollable_state: ScrollableListState {
                list_state: ListState::default(),
            },
            selected_index: Some(0),
        };

        let widget = ListDetails::from_states(
            Block::default().borders(ratatui::widgets::Borders::ALL),
            &items,
        );

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(widget, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }

    #[test]
    fn test_change_summary_details() {
        let change_summary = StackChangeSummary {
            steps: vec![
                pulumi_automation::stack::StackChangeStep {
                    op: Operation::Create,
                    urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::new-bucket".to_string(),
                    provider: None,
                    new_state: Some(ResourceState {
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::new-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        ..Default::default()
                    }),
                    old_state: None,
                    extra_values: None,
                },
                pulumi_automation::stack::StackChangeStep {
                    op: Operation::Update,
                    urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::existing-bucket"
                        .to_string(),
                    provider: None,
                    new_state: Some(ResourceState {
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::existing-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        ..Default::default()
                    }),
                    old_state: None,
                    extra_values: None,
                },
            ],
            change_summary: None,
            duration: 0,
            extra_values: None,
        };

        let mut state = ListDetailsState {
            scrollable_state: ScrollableListState {
                list_state: ListState::default(),
            },
            selected_index: Some(0),
        };

        let widget = ListDetails::from_summary(
            Block::default().borders(ratatui::widgets::Borders::ALL),
            &change_summary,
        );

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(widget, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }

    #[test]
    fn test_operation_details() {
        let operations = vec![
            ResourceOperationState::InProgress {
                sequence: 1,
                start_time: chrono::Utc::now(),
                pre_event: ResourcePreDetails {
                    metadata: StepEventMetadata {
                        op: Operation::Create,
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::my-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        old: None,
                        new: None,
                        keys: None,
                        diffs: None,
                        detailed_diff: None,
                        logical: None,
                        provider: "".to_string(),
                    },
                    planning: None,
                    extra_values: None,
                },
            },
            ResourceOperationState::Completed {
                sequence: 2,
                start_time: chrono::Utc::now(),
                end_time: chrono::Utc::now(),
                pre_event: ResourcePreDetails {
                    metadata: StepEventMetadata {
                        op: Operation::Update,
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::other-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        old: None,
                        new: None,
                        keys: None,
                        diffs: None,
                        detailed_diff: None,
                        logical: None,
                        provider: "".to_string(),
                    },
                    planning: None,
                    extra_values: None,
                },
                out_event: pulumi_automation::event::ResOutputsDetails {
                    metadata: StepEventMetadata {
                        op: Operation::Update,
                        urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::other-bucket"
                            .to_string(),
                        resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
                        old: None,
                        new: None,
                        keys: None,
                        diffs: None,
                        detailed_diff: None,
                        logical: None,
                        provider: "".to_string(),
                    },
                    planning: None,
                    extra_values: None,
                },
            },
        ];

        let mut state = ListDetailsState {
            scrollable_state: ScrollableListState {
                list_state: ListState::default(),
            },
            selected_index: Some(1),
        };

        let widget = ListDetails::from_operations(
            Block::default().borders(ratatui::widgets::Borders::ALL),
            &operations,
        );

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(widget, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }

    #[test]
    fn test_no_selection() {
        let items = vec![ResourceState {
            urn: "urn:pulumi:stack::project::aws:s3/bucket:Bucket::my-bucket".to_string(),
            resource_type: ResourceType::Other("aws:s3/bucket:Bucket".to_string()),
            ..Default::default()
        }];

        let mut state = ListDetailsState {
            scrollable_state: ScrollableListState {
                list_state: ListState::default(),
            },
            selected_index: None,
        };

        let widget = ListDetails::from_states(
            Block::default().borders(ratatui::widgets::Borders::ALL),
            &items,
        );

        let mut terminal = Terminal::new(TestBackend::new(80, 20)).unwrap();
        terminal
            .draw(|frame| frame.render_stateful_widget(widget, frame.area(), &mut state))
            .unwrap();
        assert_snapshot!(terminal.backend());
    }
}
