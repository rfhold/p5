use std::fmt::{Display, Formatter};

use p5::{AppEvent, EventType};
use ratatui::widgets::ListState;

use crate::{
    contexts::AppContextKey,
    pulumi::{self, StackResource},
    ui::MultiSelectState,
};

#[derive(Debug, Clone, Default)]
pub struct Model {
    pub selected_program: Option<Program>,
    pub program_list: ProgramList,
    pub context_order: Vec<AppContextKey>,
    pub current_context: AppContextKey,
    pub events: Vec<AppEvent>,
    pub error_backlog: Vec<AppEvent>,
}

impl Model {
    pub fn highlighted_program(&self) -> Option<Program> {
        if let Some(selected_program) = self.program_list.list_state.selected() {
            return Some(self.program_list.programs[selected_program].clone());
        }
        None
    }

    pub fn highlighted_stack(&self) -> Option<StackModel> {
        if let Some(selected_program) = self.selected_program.clone() {
            if let Some(selected_stack) = selected_program.stack_list.as_ref() {
                if let Some(selected_stack_index) = selected_stack.list_state.selected() {
                    return Some(selected_stack.stacks[selected_stack_index].clone());
                }
            }
        }
        None
    }

    pub fn highlighted_resource(&self) -> Option<ResourceModel> {
        if let Some(selected_program) = self.selected_program.clone() {
            if let Some(selected_stack) = selected_program.selected_stack {
                if let Some(selected_resource) = selected_stack.resource_list.as_ref() {
                    if let Some(selected_resource_index) = selected_resource.list_state.selected() {
                        return Some(selected_resource.resources[selected_resource_index].clone());
                    }
                }
            }
        }
        None
    }

    pub fn selected_program(&self) -> Option<Program> {
        self.selected_program.clone()
    }

    pub fn selected_stack(&self) -> Option<StackModel> {
        if let Some(selected_program) = self.selected_program.clone() {
            return selected_program.selected_stack;
        }
        None
    }

    pub fn selected_resources(&self) -> Option<Vec<ResourceModel>> {
        if let Some(selected_program) = self.selected_program.clone() {
            if let Some(selected_stack) = selected_program.selected_stack {
                return selected_stack.selected_resource_urns.as_ref().map(|urns| {
                    urns.iter()
                        .filter_map(|urn| {
                            selected_stack
                                .resource_list
                                .as_ref()
                                .and_then(|rl| rl.resources.iter().find(|r| r.resource.urn == *urn))
                        })
                        .cloned()
                        .collect()
                });
            }
        }
        None
    }

    pub fn selected_or_select_program(&mut self) -> Option<Program> {
        if let Some(selected_program) = self.selected_program() {
            return Some(selected_program);
        }
        self.select_highlighted_program()
    }

    pub fn selected_or_select_stack(&mut self) -> Option<StackModel> {
        if let Some(selected_stack) = self.selected_stack() {
            return Some(selected_stack);
        }
        self.select_highlighted_stack()
    }

    pub fn select_highlighted_program(&mut self) -> Option<Program> {
        if let Some(selected_program) = self.highlighted_program() {
            self.selected_program = Some(selected_program.clone());
            return Some(selected_program);
        }
        None
    }

    pub fn select_highlighted_stack(&mut self) -> Option<StackModel> {
        if let Some(selected_stack) = self.highlighted_stack() {
            self.selected_program.as_mut().unwrap().selected_stack = Some(selected_stack.clone());
            return Some(selected_stack);
        }
        None
    }

    pub fn toggle_highlighted_resource(&mut self) {
        if let Some(selected_stack) = self.selected_stack() {
            if let Some(selected_resource) = self.highlighted_resource() {
                if let Some(selected_urns) = selected_stack.selected_resource_urns.clone() {
                    if selected_urns.contains(&selected_resource.resource.urn) {
                        self.selected_program
                            .as_mut()
                            .unwrap()
                            .selected_stack
                            .as_mut()
                            .unwrap()
                            .selected_resource_urns
                            .as_mut()
                            .unwrap()
                            .retain(|urn| urn != &selected_resource.resource.urn);
                    } else {
                        self.selected_program
                            .as_mut()
                            .unwrap()
                            .selected_stack
                            .as_mut()
                            .unwrap()
                            .selected_resource_urns
                            .as_mut()
                            .unwrap()
                            .push(selected_resource.resource.urn.clone());
                    }
                }
            }
        }
    }

    pub fn select_next_program(&mut self) {
        self.program_list.list_state.select_next();
    }

    pub fn select_previous_program(&mut self) {
        self.program_list.list_state.select_previous();
    }

    pub fn select_next_stack(&mut self) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(list) = &mut selected_program.stack_list {
                list.list_state.select_next();
            }
        }
    }

    pub fn select_previous_stack(&mut self) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(list) = &mut selected_program.stack_list {
                list.list_state.select_previous();
            }
        }
    }

    pub fn select_next_resource(&mut self) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(selected_stack) = &mut selected_program.selected_stack {
                if let Some(list) = &mut selected_stack.resource_list {
                    list.list_state.select_next();
                }
            }
        }
    }

    pub fn select_previous_resource(&mut self) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(selected_stack) = &mut selected_program.selected_stack {
                if let Some(list) = &mut selected_stack.resource_list {
                    list.list_state.select_previous();
                }
            }
        }
    }

    pub fn stack_list(&self) -> Option<StackList> {
        if let Some(selected_program) = self.selected_program.clone() {
            return selected_program.stack_list;
        }
        None
    }

    pub fn resource_list(&self) -> Option<ResourceList> {
        if let Some(selected_program) = self.selected_program.clone() {
            if let Some(selected_stack) = selected_program.selected_stack {
                return selected_stack.resource_list;
            }
        }
        None
    }

    pub fn set_selected_stack_output(
        &mut self,
        output: Option<serde_json::Value>,
        error: Option<pulumi::CommandError>,
    ) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(selected_stack) = &mut selected_program.selected_stack {
                selected_stack.view = StackView::Output {
                    output,
                    command_error: error,
                };
            }
        }
    }

    pub fn set_selected_stack_view(&mut self, view: Option<StackView>) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(selected_stack) = &mut selected_program.selected_stack {
                selected_stack.view = view.unwrap_or(StackView::None);
            }
        }
    }

    pub fn focus_next_context(&mut self) {
        let current_index = self
            .context_order
            .iter()
            .position(|x| x == &self.current_context)
            .unwrap_or(0);
        let next_index = (current_index + 1) % self.context_order.len();
        self.current_context = self.context_order[next_index].clone();
    }

    pub fn focus_previous_context(&mut self) {
        let current_index = self
            .context_order
            .iter()
            .position(|x| x == &self.current_context)
            .unwrap_or(0);
        let previous_index = if current_index == 0 {
            self.context_order.len() - 1
        } else {
            current_index - 1
        };
        self.current_context = self.context_order[previous_index].clone();
    }

    pub fn add_event(&mut self, event: AppEvent) {
        if event.event_type == EventType::Error {
            self.error_backlog.push(event.clone());
        }

        self.events.push(event);
    }

    pub fn set_resource_list(&mut self, resource_list: Option<ResourceList>) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(selected_stack) = &mut selected_program.selected_stack {
                selected_stack.resource_list = resource_list;
            }
        }
    }

    pub fn set_stack_list(&mut self, stack_list: Option<StackList>) {
        if let Some(selected_program) = &mut self.selected_program {
            selected_program.stack_list = stack_list;
        }
    }

    pub fn select_next_operation_view_item(&mut self) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(selected_stack) = &mut selected_program.selected_stack {
                selected_stack.view.select_next();
            }
        }
    }

    pub fn select_previous_operation_view_item(&mut self) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(selected_stack) = &mut selected_program.selected_stack {
                selected_stack.view.select_previous();
            }
        }
    }

    pub fn toggle_operation_view_item(&mut self) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(selected_stack) = &mut selected_program.selected_stack {
                selected_stack.view.toggle();
            }
        }
    }

    pub fn focus_operation_view_item(&mut self) {
        if let Some(selected_program) = &mut self.selected_program {
            if let Some(selected_stack) = &mut selected_program.selected_stack {
                selected_stack.view.focus_current();
            }
        }
    }
}

#[derive(Debug, Clone, Default)]
pub struct ProgramList {
    pub programs: Vec<Program>,
    pub list_state: ListState,
}

#[derive(Debug, Clone)]
pub struct Program {
    pub path: String,
    pub name: String,
    pub selected_stack: Option<StackModel>,
    pub stack_list: Option<StackList>,
}

impl Display for Program {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.name)
    }
}

#[derive(Debug, Clone, Default)]
pub struct StackList {
    pub stacks: Vec<StackModel>,
    pub list_state: ListState,
}

#[derive(Debug, Clone, Default)]
pub struct StackModel {
    pub stack: pulumi::Stack,
    pub selected_resource_urns: Option<Vec<String>>,
    pub resource_list: Option<ResourceList>,

    pub view: StackView,
}

#[derive(Debug, Clone, Default)]
pub enum StackView {
    #[default]
    None,
    Output {
        output: Option<serde_json::Value>,
        command_error: Option<pulumi::CommandError>,
    },
    Preview {
        steps: Option<Vec<StepViewModel>>,
        command_error: Option<pulumi::CommandError>,
        state: MultiSelectState,
    },
}

impl StackView {
    pub fn select_next(&mut self) {
        if let StackView::Preview { state, .. } = self {
            state.list_state.select_next();
        }
    }

    pub fn select_previous(&mut self) {
        if let StackView::Preview { state, .. } = self {
            state.list_state.select_previous();
        }
    }

    pub fn toggle(&mut self) {
        if let StackView::Preview { state, .. } = self {
            state.toggle();
        }
    }

    fn focus_current(&mut self) {
        if let StackView::Preview { steps, state, .. } = self {
            if let Some(selected_index) = state.list_state.selected() {
                if let Some(step) = steps.as_mut().and_then(|s| s.get_mut(selected_index)) {
                    step.expanded = !step.expanded;
                }
            }
        }
    }
}

impl Display for StackModel {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.stack.name)
    }
}

#[derive(Debug, Clone, Default)]
pub struct StepViewModel {
    pub step: pulumi::Step,
    pub expanded: bool,
}

#[derive(Debug, Clone, Default)]
pub struct ResourceList {
    pub resources: Vec<ResourceModel>,
    pub list_state: ListState,
}

#[derive(Debug, Clone)]
pub struct ResourceModel {
    pub resource: StackResource,
}

impl Display for ResourceModel {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.get_name())
    }
}

impl ResourceModel {
    pub fn get_name(&self) -> String {
        return self
            .resource
            .urn
            .split("::")
            .last()
            .expect("Invalid URN when getting name")
            .to_string();
    }
}
