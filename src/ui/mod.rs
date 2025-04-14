use ratatui::widgets::{ListItem, ListState};

use crate::model::{Program, ResourceModel, StackModel};

mod json;
pub mod layout;
mod panel;
mod preview;
mod view;

impl<'a> Into<ListItem<'a>> for Program {
    fn into(self) -> ListItem<'a> {
        ListItem::new(self.name)
    }
}

impl<'a> Into<ListItem<'a>> for StackModel {
    fn into(self) -> ListItem<'a> {
        ListItem::new(self.stack.name)
    }
}

impl<'a> Into<ListItem<'a>> for ResourceModel {
    fn into(self) -> ListItem<'a> {
        ListItem::new(self.get_name())
    }
}

#[derive(Debug, Clone, Default)]
pub struct MultiSelectState {
    pub selected: Vec<usize>,
    pub list_state: ListState,
}

impl MultiSelectState {
    pub fn toggle(&mut self) {
        let index = self.list_state.selected().unwrap_or(0);

        if self.selected.contains(&index) {
            self.selected.retain(|&x| x != index);
        } else {
            self.selected.push(index);
        }
    }

    pub fn is_selected(&self, index: usize) -> bool {
        self.selected.contains(&index)
    }
}
