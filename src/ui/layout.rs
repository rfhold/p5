use ratatui::{
    buffer::Buffer,
    layout::{self, Rect},
    widgets::{ListState, StatefulWidget, Widget},
};

use crate::{
    contexts::AppContextKey,
    model::{Model, StackView},
};

use super::{panel::SelectionPanel, view::View};

#[derive(Debug, Clone)]
pub struct Layout {}

impl Layout {
    pub fn new() -> Self {
        Layout {}
    }
}

impl StatefulWidget for Layout {
    type State = Model;

    fn render(self, area: Rect, buf: &mut Buffer, state: &mut Self::State) {
        let main_layout = layout::Layout::default()
            .direction(layout::Direction::Horizontal)
            .constraints(
                [
                    layout::Constraint::Percentage(30),
                    layout::Constraint::Percentage(70),
                ]
                .as_ref(),
            )
            .split(area);

        let left_area = main_layout[0];
        let right_area = main_layout[1];

        let left_vertical_layout = layout::Layout::default()
            .direction(layout::Direction::Vertical)
            .constraints(
                [
                    layout::Constraint::Percentage(40),
                    layout::Constraint::Percentage(60),
                ]
                .as_ref(),
            )
            .split(left_area);

        let program_area = left_vertical_layout[0];
        let stack_area = left_vertical_layout[1];

        let right_vertical_layout = layout::Layout::default()
            .direction(layout::Direction::Vertical)
            .constraints(
                [
                    layout::Constraint::Percentage(80),
                    layout::Constraint::Percentage(20),
                ]
                .as_ref(),
            )
            .split(right_area);

        let output_area = right_vertical_layout[0];
        let status_area = right_vertical_layout[1];

        let panel = SelectionPanel::new(Some(&state.program_list.programs));

        panel
            .with_title("Programs".to_string())
            .with_active(state.current_context == AppContextKey::Programs)
            .render(program_area, buf, &mut state.program_list.list_state);

        let stacks = if let Some(selected_program) = state.selected_program.clone() {
            if let Some(stack_list) = selected_program.stack_list {
                stack_list.stacks
            } else {
                vec![]
            }
        } else {
            vec![]
        };

        let stack_panel = SelectionPanel::new(Some(&stacks));

        let mut stack_list_state = match state.stack_list() {
            Some(_) => {
                &mut state
                    .selected_program
                    .as_mut()
                    .unwrap()
                    .stack_list
                    .as_mut()
                    .unwrap()
                    .list_state
            }
            None => &mut ListState::default(),
        };

        stack_panel
            .with_title("Stacks".to_string())
            .with_active(state.current_context == AppContextKey::Stacks)
            .render(stack_area, buf, &mut stack_list_state);

        SelectionPanel::new(Some(&state.events))
            .with_active(state.current_context == AppContextKey::Status)
            .with_title("Status".to_string())
            .render(status_area, buf, &mut ListState::default());

        if let Some(_selected_stack) = &state.selected_stack() {
            let selected_stack = state
                .selected_program
                .as_mut()
                .unwrap()
                .selected_stack
                .as_mut()
                .unwrap();

            View::default()
                .with_active(state.current_context == AppContextKey::OperationView)
                .render(output_area, buf, &mut selected_stack.view);
        }
    }
}
