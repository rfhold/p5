use pulumi_automation::event::DiffKind;
use pulumi_automation::stack::{ResourceState, StackChangeSummary};
use ratatui::{
    buffer::Buffer,
    layout::Rect,
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, ListItem, StatefulWidget},
};
use ratatui_macros::{line, span};
use std::collections::HashMap;

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

        // Add inputs section with actual values
        if let Some(ref inputs) = resource.inputs {
            details.push(DetailItem {
                label: String::new(),
                line: line![span!["── Inputs ──"].style(Style::default().fg(Color::Cyan))],
            });
            details.extend(format_json_value(inputs, 2, 60));
        }

        // Add outputs section with actual values
        if let Some(ref outputs) = resource.outputs {
            details.push(DetailItem {
                label: String::new(),
                line: line![span!["── Outputs ──"].style(Style::default().fg(Color::Cyan))],
            });
            details.extend(format_json_value(outputs, 2, 60));
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

        let mut details = vec![
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
        ];

        // Show what's changing
        match (&step.old_state, &step.new_state) {
            (Some(old), Some(new)) => {
                // For updates, show property changes
                details.push(DetailItem {
                    label: String::new(),
                    line: line![span!["── Changes ──"].style(Style::default().fg(Color::Cyan))],
                });

                // Compare inputs if both exist
                if let (Some(old_inputs), Some(new_inputs)) = (&old.inputs, &new.inputs) {
                    let changes = compare_json_values(old_inputs, new_inputs, "");
                    details.extend(changes);
                }
            }
            (None, Some(new)) => {
                // For creates, show new properties
                if let Some(inputs) = &new.inputs {
                    details.push(DetailItem {
                        label: String::new(),
                        line: line![
                            span!["── New Resource ──"].style(Style::default().fg(Color::Green))
                        ],
                    });
                    details.extend(format_json_value(inputs, 2, 60));
                }
            }
            (Some(old), None) => {
                // For deletes, show old properties
                if let Some(inputs) = &old.inputs {
                    details.push(DetailItem {
                        label: String::new(),
                        line: line![
                            span!["── Deleted Resource ──"].style(Style::default().fg(Color::Red))
                        ],
                    });
                    details.extend(format_json_value(inputs, 2, 60));
                }
            }
            _ => {}
        }

        details
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

        // Show detailed diff if available
        if let Some(ref detailed_diff) = metadata.detailed_diff {
            details.push(DetailItem {
                label: String::new(),
                line: line![
                    span!["── Property Changes ──"].style(Style::default().fg(Color::Cyan))
                ],
            });

            for (property, diff) in detailed_diff {
                details.push(DetailItem {
                    label: "  ".to_string(),
                    line: line![diff_kind_to_span(&diff.diff_kind, property)],
                });
            }
        }

        // Show actual changes
        match (&metadata.old, &metadata.new) {
            (Some(old), Some(new)) => {
                details.push(DetailItem {
                    label: String::new(),
                    line: line![
                        span!["── Detailed Changes ──"].style(Style::default().fg(Color::Cyan))
                    ],
                });

                let changes = compare_json_maps(&old.inputs, &new.inputs, "");
                details.extend(changes);
            }
            (None, Some(new)) => {
                details.push(DetailItem {
                    label: String::new(),
                    line: line![
                        span!["── New Properties ──"].style(Style::default().fg(Color::Green))
                    ],
                });

                for (key, value) in &new.inputs {
                    details.push(DetailItem {
                        label: format!("  {}", key),
                        line: line![
                            span![format_value_summary(value)]
                                .style(Style::default().fg(Color::Green))
                        ],
                    });
                }
            }
            (Some(old), None) => {
                details.push(DetailItem {
                    label: String::new(),
                    line: line![
                        span!["── Deleted Properties ──"].style(Style::default().fg(Color::Red))
                    ],
                });

                for (key, value) in &old.inputs {
                    details.push(DetailItem {
                        label: format!("  {}", key),
                        line: line![
                            span![format_value_summary(value)]
                                .style(Style::default().fg(Color::Red))
                        ],
                    });
                }
            }
            _ => {}
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

fn diff_kind_to_span(diff_kind: &DiffKind, property: &str) -> Span<'static> {
    let (prefix, color) = match diff_kind {
        DiffKind::Add => ("+ ", Color::Green),
        DiffKind::AddReplace => ("± ", Color::Magenta),
        DiffKind::Delete => ("- ", Color::Red),
        DiffKind::DeleteReplace => ("∓ ", Color::Magenta),
        DiffKind::Update => ("~ ", Color::Yellow),
        DiffKind::UpdateReplace => ("≈ ", Color::Magenta),
    };
    let text = match diff_kind {
        DiffKind::AddReplace | DiffKind::DeleteReplace | DiffKind::UpdateReplace => {
            format!("{}{} (replace)", prefix, property)
        }
        _ => format!("{}{}", prefix, property),
    };
    span![text].style(Style::default().fg(color))
}

fn format_value_summary(value: &serde_json::Value) -> String {
    match value {
        serde_json::Value::Null => "null".to_string(),
        serde_json::Value::Bool(b) => b.to_string(),
        serde_json::Value::Number(n) => n.to_string(),
        serde_json::Value::String(s) => {
            if s.starts_with("[secret]") || s.contains("ciphertext") {
                "[secret]".to_string()
            } else if s.len() > 50 {
                format!("\"{}...\"", &s[..47])
            } else {
                format!("\"{}\"", s)
            }
        }
        serde_json::Value::Array(arr) => format!("[{} items]", arr.len()),
        serde_json::Value::Object(obj) => {
            if obj.contains_key("ciphertext") {
                "[secret]".to_string()
            } else {
                format!("{{{} props}}", obj.len())
            }
        }
    }
}

fn format_json_value(
    value: &serde_json::Value,
    indent: usize,
    max_width: usize,
) -> Vec<DetailItem> {
    let mut items = Vec::new();

    match value {
        serde_json::Value::Object(map) => {
            for (key, val) in map {
                let label = format!("{}{}", " ".repeat(indent), key);
                let value_str = format_value_summary(val);

                // Handle nested objects and arrays
                match val {
                    serde_json::Value::Object(_) | serde_json::Value::Array(_) => {
                        items.push(DetailItem {
                            label: label.clone(),
                            line: line![span![value_str]],
                        });
                        // Optionally expand nested structures
                        if indent < 6 && (val.is_object() || val.is_array()) {
                            items.extend(format_json_value(val, indent + 2, max_width));
                        }
                    }
                    _ => {
                        items.push(DetailItem {
                            label,
                            line: line![span![value_str]],
                        });
                    }
                }
            }
        }
        serde_json::Value::Array(arr) => {
            for (i, val) in arr.iter().enumerate() {
                let label = format!("{}[{}]", " ".repeat(indent), i);
                items.push(DetailItem {
                    label,
                    line: line![span![format_value_summary(val)]],
                });
            }
        }
        _ => {
            items.push(DetailItem {
                label: " ".repeat(indent),
                line: line![span![format_value_summary(value)]],
            });
        }
    }

    items
}

fn compare_json_values<'a>(
    old: &'a serde_json::Value,
    new: &'a serde_json::Value,
    path: &'a str,
) -> Vec<DetailItem<'a>> {
    match (old, new) {
        (serde_json::Value::Object(old_map), serde_json::Value::Object(new_map)) => {
            compare_json_maps_from_serde(old_map, new_map, path)
        }
        _ => {
            // For non-objects, just show the change
            vec![DetailItem {
                label: path.to_string(),
                line: line![
                    span![format_value_summary(old)].style(Style::default().fg(Color::Red)),
                    span![" → "],
                    span![format_value_summary(new)].style(Style::default().fg(Color::Green))
                ],
            }]
        }
    }
}

fn compare_json_maps_from_serde<'a>(
    old: &'a serde_json::Map<String, serde_json::Value>,
    new: &'a serde_json::Map<String, serde_json::Value>,
    prefix: &'a str,
) -> Vec<DetailItem<'a>> {
    let mut items = Vec::new();
    let indent = if prefix.is_empty() {
        2
    } else {
        prefix.len() + 2
    };

    // Check for deleted properties
    for (key, old_value) in old {
        if !new.contains_key(key) {
            items.push(DetailItem {
                label: format!("{}{}", " ".repeat(indent), key),
                line: line![
                    span!["- "].style(Style::default().fg(Color::Red)),
                    span![format_value_summary(old_value)]
                ],
            });
        }
    }

    // Check for added or modified properties
    for (key, new_value) in new {
        if let Some(old_value) = old.get(key) {
            if old_value != new_value {
                // Modified
                items.push(DetailItem {
                    label: format!("{}{}", " ".repeat(indent), key),
                    line: line![
                        span!["~ "].style(Style::default().fg(Color::Yellow)),
                        span![format_value_summary(old_value)]
                            .style(Style::default().fg(Color::Red)),
                        span![" → "],
                        span![format_value_summary(new_value)]
                            .style(Style::default().fg(Color::Green))
                    ],
                });
            }
        } else {
            // Added
            items.push(DetailItem {
                label: format!("{}{}", " ".repeat(indent), key),
                line: line![
                    span!["+ "].style(Style::default().fg(Color::Green)),
                    span![format_value_summary(new_value)]
                ],
            });
        }
    }

    items
}

fn compare_json_maps<'a>(
    old: &'a HashMap<String, serde_json::Value>,
    new: &'a HashMap<String, serde_json::Value>,
    prefix: &'a str,
) -> Vec<DetailItem<'a>> {
    let mut items = Vec::new();
    let indent = if prefix.is_empty() {
        2
    } else {
        prefix.len() + 2
    };

    // Check for deleted properties
    for (key, old_value) in old {
        if !new.contains_key(key) {
            items.push(DetailItem {
                label: format!("{}{}", " ".repeat(indent), key),
                line: line![
                    span!["- "].style(Style::default().fg(Color::Red)),
                    span![format_value_summary(old_value)]
                ],
            });
        }
    }

    // Check for added or modified properties
    for (key, new_value) in new {
        if let Some(old_value) = old.get(key) {
            if old_value != new_value {
                // Modified
                items.push(DetailItem {
                    label: format!("{}{}", " ".repeat(indent), key),
                    line: line![
                        span!["~ "].style(Style::default().fg(Color::Yellow)),
                        span![format_value_summary(old_value)]
                            .style(Style::default().fg(Color::Red)),
                        span![" → "],
                        span![format_value_summary(new_value)]
                            .style(Style::default().fg(Color::Green))
                    ],
                });
            }
        } else {
            // Added
            items.push(DetailItem {
                label: format!("{}{}", " ".repeat(indent), key),
                line: line![
                    span!["+ "].style(Style::default().fg(Color::Green)),
                    span![format_value_summary(new_value)]
                ],
            });
        }
    }

    items
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
