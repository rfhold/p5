use crossterm::event::Event;
use tui_input::{Input, backend::crossterm::EventHandler};

pub enum InputResult {
    Submit(String),
    Cancel,
}

pub fn handle_user_input(event: Event, input: &mut Input) -> Option<InputResult> {
    return if let Event::Key(key) = event {
        match key.code {
            crossterm::event::KeyCode::Esc => Some(InputResult::Cancel),
            crossterm::event::KeyCode::Enter => Some(InputResult::Submit(input.to_string())),
            _ => {
                input.handle_event(&event);
                None
            }
        }
    } else {
        None
    };
}
