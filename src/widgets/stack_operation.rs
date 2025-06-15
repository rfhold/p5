use ratatui::{
    layout::{Alignment, Direction, Layout},
    style::Style,
    widgets::{Block, Paragraph, StatefulWidget, Widget},
};

use crate::state::{
    AppContext, AppState, Loadable, OperationContext, ProgramOperation, StackContext,
};

use super::{ResourceListState, resource_list::ResourceList, theme::color};

#[derive(Default, Clone)]
pub struct StackOperation {}

#[derive(Debug, Clone, strum::Display, Eq, PartialEq)]
enum StackOperationType {
    Update,
    Refresh,
    Destroy,
    Preview,
}

impl StatefulWidget for StackOperation {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let background_context = state.background_context();
        if let Some((operation, selection)) = state.stack_operation_state() {
            let change_summary = operation.change_summary.as_ref();
            let events = operation.events.as_ref();
            let operation_view = match operation.operation {
                ProgramOperation::Update if operation.is_preview() => StackOperationType::Preview,
                ProgramOperation::Update => StackOperationType::Update,
                ProgramOperation::Refresh => StackOperationType::Refresh,
                ProgramOperation::Destroy => StackOperationType::Destroy,
            };

            let title = format!(
                "{} - {}",
                operation_view,
                match change_summary {
                    Loadable::Loaded(_) => match events {
                        Loadable::Loaded(e) => {
                            if e.done { "Complete" } else { "In Progress" }
                        }
                        Loadable::Loading => "Loading Events",
                        Loadable::NotLoaded => "Waiting for Events",
                    },
                    Loadable::Loading => "Loading Summary",
                    Loadable::NotLoaded => "No Summary Available",
                }
            );

            let should_split = events.is_loaded() && operation_view != StackOperationType::Preview;

            let layout_constraints = if should_split {
                vec![
                    ratatui::layout::Constraint::Percentage(90),
                    ratatui::layout::Constraint::Percentage(10),
                ]
            } else {
                vec![ratatui::layout::Constraint::Percentage(100)]
            };

            let layout = Layout::default()
                .direction(Direction::Vertical)
                .constraints(layout_constraints)
                .split(area);

            let summary_area = if should_split { layout[1] } else { layout[0] };

            if let Loadable::Loaded(events) = events {
                ResourceList::from_operations(
                    Block::bordered()
                        .title(title.to_string())
                        .border_type(ratatui::widgets::BorderType::Rounded)
                        .border_style(Style::default().fg(
                            if let AppContext::Stack(StackContext::Operation(
                                OperationContext::Events,
                            )) = background_context
                            {
                                color::BORDER_HIGHLIGHT
                            } else {
                                color::BORDER_DEFAULT
                            },
                        )),
                    &events.states,
                )
                .render(layout[0], buf, &mut ResourceListState::default());
            }

            let block = Block::bordered()
                .title(if events.is_loaded() {
                    "Preview - Summary".to_string()
                } else {
                    "Preview".to_string()
                })
                .border_type(ratatui::widgets::BorderType::Rounded)
                .border_style(Style::default().fg(
                    if let AppContext::Stack(StackContext::Operation(OperationContext::Summary)) =
                        background_context
                    {
                        color::BORDER_HIGHLIGHT
                    } else {
                        color::BORDER_DEFAULT
                    },
                ));

            match change_summary {
                Loadable::Loaded(change_summary) => {
                    ResourceList::from_summary(block, change_summary).render(
                        summary_area,
                        buf,
                        selection,
                    );
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
        } else {
            Paragraph::new("No Operation in Progress")
                .block(
                    Block::bordered()
                        .title("Stack Operation")
                        .border_type(ratatui::widgets::BorderType::Rounded),
                )
                .alignment(Alignment::Center)
                .render(area, buf);
        }
    }
}
