use ratatui::{
    layout::{Alignment, Constraint, Direction, Layout},
    style::Style,
    widgets::{Block, Paragraph, StatefulWidget, Widget},
};

use crate::state::{
    AppContext, AppState, Loadable, OperationContext, OperationDetailsContent, OperationProgress,
    ProgramOperation, StackContext,
};

use super::{
    ResourceListState,
    list_details::{ListDetails, ListDetailsState},
    resource_list::ResourceList,
    theme::color,
};

#[derive(Default, Clone)]
pub struct StackOperation {}

#[derive(Debug, Clone, strum::Display, Eq, PartialEq)]
enum StackOperationType {
    Update,
    Refresh,
    Destroy,
    Preview,
}

impl From<OperationProgress> for StackOperationType {
    fn from(progress: OperationProgress) -> Self {
        match progress.operation {
            ProgramOperation::Update if progress.is_preview() => StackOperationType::Preview,
            ProgramOperation::Update => StackOperationType::Update,
            ProgramOperation::Refresh => StackOperationType::Refresh,
            ProgramOperation::Destroy => StackOperationType::Destroy,
        }
    }
}

impl StatefulWidget for StackOperation {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let current_context = state.current_context();
        let background_context = state.background_context();

        // Determine the current view mode
        let view_mode = match &current_context {
            AppContext::Stack(StackContext::Operation(OperationContext::Summary(mode))) => mode,
            AppContext::Stack(StackContext::Operation(OperationContext::Events(mode))) => mode,
            _ => &OperationDetailsContent::List,
        };

        if let Some((operation, selection)) = state.stack_operation_state() {
            let operation_view = StackOperationType::from(operation.clone());
            let events = operation.events.as_ref();
            let change_summary = operation.change_summary.as_ref();

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

            // Check if we should show details panel
            let show_details = matches!(
                view_mode,
                OperationDetailsContent::Details | OperationDetailsContent::Item
            );

            // Split layout horizontally if in Details or Item mode
            let main_layout = if show_details {
                Layout::default()
                    .direction(Direction::Horizontal)
                    .constraints([Constraint::Percentage(50), Constraint::Percentage(50)])
                    .split(area)
            } else {
                // Return a single-element layout for consistency
                Layout::default()
                    .direction(Direction::Horizontal)
                    .constraints([Constraint::Percentage(100)])
                    .split(area)
            };

            let list_area = main_layout[0];
            let details_area = if show_details {
                Some(main_layout[1])
            } else {
                None
            };

            // Determine if we're showing events or summary
            let showing_events = matches!(
                current_context,
                AppContext::Stack(StackContext::Operation(OperationContext::Events(_)))
            );

            if showing_events {
                // Show events list
                let should_split =
                    events.is_loaded() && operation_view != StackOperationType::Preview;

                let layout_constraints = if should_split {
                    vec![Constraint::Percentage(90), Constraint::Percentage(10)]
                } else {
                    vec![Constraint::Percentage(100)]
                };

                let layout = Layout::default()
                    .direction(Direction::Vertical)
                    .constraints(layout_constraints)
                    .split(list_area);

                let summary_area = if should_split { layout[1] } else { layout[0] };

                if let Loadable::Loaded(events) = events {
                    ResourceList::from_operations(
                        Block::bordered()
                            .title(title.to_string())
                            .border_type(ratatui::widgets::BorderType::Rounded)
                            .border_style(Style::default().fg(
                                if let AppContext::Stack(StackContext::Operation(
                                    OperationContext::Events(_),
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

                    // Render details panel if in Details or Item mode
                    if let Some(details_area) = details_area {
                        let selected_index = selection.scrollable_state.list_state.selected();

                        // Ensure the selected index is valid for the current events data
                        let valid_selected_index = match selected_index {
                            Some(idx) if idx < events.states.len() => Some(idx),
                            Some(idx) => {
                                #[cfg(debug_assertions)]
                                eprintln!(
                                    "Warning: Selected index {} exceeds events count {}",
                                    idx,
                                    events.states.len()
                                );
                                None
                            }
                            None => None,
                        };

                        #[cfg(debug_assertions)]
                        eprintln!(
                            "Events - Selected index: {:?}, Valid index: {:?}, Events count: {}",
                            selected_index,
                            valid_selected_index,
                            events.states.len()
                        );

                        let mut details_state = ListDetailsState {
                            scrollable_state: Default::default(),
                            selected_index: valid_selected_index,
                        };
                        ListDetails::from_operations(
                            Block::bordered()
                                .title("Operation Details")
                                .border_type(ratatui::widgets::BorderType::Rounded),
                            &events.states,
                        )
                        .render(details_area, buf, &mut details_state);
                    }
                }

                let block = Block::bordered()
                    .title(if events.is_loaded() {
                        "Preview - Summary".to_string()
                    } else {
                        "Preview".to_string()
                    })
                    .border_type(ratatui::widgets::BorderType::Rounded)
                    .border_style(Style::default().fg(
                        if let AppContext::Stack(StackContext::Operation(
                            OperationContext::Summary(_),
                        )) = background_context
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
                // Show summary list
                let block = Block::bordered()
                    .title("Preview - Summary")
                    .border_type(ratatui::widgets::BorderType::Rounded)
                    .border_style(Style::default().fg(
                        if let AppContext::Stack(StackContext::Operation(
                            OperationContext::Summary(_),
                        )) = background_context
                        {
                            color::BORDER_HIGHLIGHT
                        } else {
                            color::BORDER_DEFAULT
                        },
                    ));

                match change_summary {
                    Loadable::Loaded(change_summary) => {
                        ResourceList::from_summary(block, change_summary)
                            .render(list_area, buf, selection);

                        // Render details panel if in Details or Item mode
                        if let Some(details_area) = details_area {
                            let selected_index = selection.scrollable_state.list_state.selected();

                            // Ensure the selected index is valid for the current summary data
                            let valid_selected_index = match selected_index {
                                Some(idx) if idx < change_summary.steps.len() => Some(idx),
                                Some(idx) => {
                                    #[cfg(debug_assertions)]
                                    eprintln!(
                                        "Warning: Selected index {} exceeds summary steps count {}",
                                        idx,
                                        change_summary.steps.len()
                                    );
                                    None
                                }
                                None => None,
                            };

                            #[cfg(debug_assertions)]
                            eprintln!(
                                "Summary - Selected index: {:?}, Valid index: {:?}, Steps count: {}",
                                selected_index,
                                valid_selected_index,
                                change_summary.steps.len()
                            );

                            let mut details_state = ListDetailsState {
                                scrollable_state: Default::default(),
                                selected_index: valid_selected_index,
                            };
                            ListDetails::from_summary(
                                Block::bordered()
                                    .title("Summary Details")
                                    .border_type(ratatui::widgets::BorderType::Rounded),
                                change_summary,
                            )
                            .render(
                                details_area,
                                buf,
                                &mut details_state,
                            );
                        }
                    }
                    Loadable::Loading => {
                        Paragraph::new("Loading...".to_string())
                            .block(block)
                            .alignment(Alignment::Left)
                            .render(list_area, buf);
                    }
                    Loadable::NotLoaded => {
                        Paragraph::new("No Stack Selected".to_string())
                            .block(block)
                            .alignment(Alignment::Left)
                            .render(list_area, buf);
                    }
                };
            }
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
