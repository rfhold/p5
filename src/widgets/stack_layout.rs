use ratatui::{
    layout::{Alignment, Constraint, Direction, Layout},
    widgets::{Block, Paragraph, StatefulWidget, Widget},
};

use crate::state::{AppState, Loadable, StackContext};

use super::{
    stack_config::StackConfig, stack_outputs::StackOutputs, stack_preview::StackPreview,
    stack_resources::StackResources, stack_update::StackUpdate,
};

pub struct StackLayout {
    pub context: StackContext,
}

impl StackLayout {
    pub fn new(context: StackContext) -> Self {
        Self { context }
    }
}

impl StatefulWidget for StackLayout {
    type State = AppState;

    fn render(
        self,
        area: ratatui::prelude::Rect,
        buf: &mut ratatui::prelude::Buffer,
        state: &mut Self::State,
    ) {
        let workspace = state.workspace();
        let stack = state.stack();

        let title = match workspace {
            Loadable::Loaded(workspace) => format!("Workspace: {}", workspace.cwd),
            Loadable::Loading => "Loading Workspace...".to_string(),
            Loadable::NotLoaded => "No Workspace Loaded".to_string(),
        };

        let body = match stack {
            Loadable::Loaded(stack) => format!("Stack: {}", stack.name),
            Loadable::Loading => "Loading Stack...".to_string(),
            Loadable::NotLoaded => "No Stack Selected".to_string(),
        };

        let title_block = Block::bordered()
            .title(title)
            .border_type(ratatui::widgets::BorderType::Rounded);

        let body_paragraph = Paragraph::new(body)
            .block(title_block.clone())
            .alignment(Alignment::Left);

        let layout = Layout::default()
            .direction(Direction::Vertical)
            .constraints(vec![Constraint::Length(3), Constraint::Percentage(90)])
            .split(area);

        body_paragraph.render(layout[0], buf);

        match self.context {
            StackContext::Outputs => StackOutputs::default().render(layout[1], buf, state),
            StackContext::Config => StackConfig::default().render(layout[1], buf, state),
            StackContext::Resources => StackResources::default().render(layout[1], buf, state),
            StackContext::Preview => StackPreview::default().render(layout[1], buf, state),
            StackContext::Update => StackUpdate::default().render(layout[1], buf, state),
        }
    }
}
