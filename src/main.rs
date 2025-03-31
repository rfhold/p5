use std::time::Duration;

use crossterm::event::{self, Event, KeyCode};
use ratatui::Frame;
use ratatui::prelude::*;

use clap::{Parser, arg, command};

#[derive(Parser)] // requires `derive` feature
#[command(name = "p5")]
#[command(bin_name = "p5")]
struct P5 {
    #[arg(long, short, env = "PULUMI_BACKEND")]
    backend: Option<String>,
    #[arg(
        long,
        short,
        env = "PULUMI_SECRETS_PROVIDER",
        default_value = "passphrase"
    )]
    secrets_provider: Option<String>,
}

fn main() -> color_eyre::Result<()> {
    let _args = P5::parse();

    tui::install_panic_hook();
    let mut terminal = tui::init_terminal()?;
    let mut model = Model::new();

    while model.running_state != RunningState::Done {
        // Render the current view
        terminal.draw(|f| view(&mut model, f))?;

        // Handle events and map to a Message
        let mut current_msg = handle_event(&model)?;

        // Process updates as long as they return a non-None message
        while current_msg.is_some() {
            current_msg = update(&mut model, current_msg.unwrap());
        }
    }

    tui::restore_terminal()?;
    Ok(())
}

struct Model {
    running_state: RunningState,
}

impl Model {
    fn new() -> Self {
        Self {
            running_state: RunningState::default(),
        }
    }
}

#[derive(Debug, Default, PartialEq, Eq)]
enum RunningState {
    #[default]
    Boot,
    Running,
    Done,
}

#[derive(PartialEq)]
enum Message {
    Boot,
    Quit,
    Yes,
}

fn view(_model: &mut Model, frame: &mut Frame) {
    let _main_layout = Layout::default()
        .direction(Direction::Horizontal)
        .constraints(vec![Constraint::Percentage(30), Constraint::Percentage(70)])
        .split(frame.area());
}

fn handle_event(model: &Model) -> color_eyre::Result<Option<Message>> {
    if model.running_state == RunningState::Boot {
        return Ok(Some(Message::Boot));
    }

    if event::poll(Duration::from_millis(250))? {
        if let Event::Key(key) = event::read()? {
            if key.kind == event::KeyEventKind::Press {
                return Ok(handle_key(key));
            }
        }
    }
    Ok(None)
}

fn handle_key(key: event::KeyEvent) -> Option<Message> {
    let ctrl = key.modifiers.contains(event::KeyModifiers::CONTROL);
    match key.code {
        KeyCode::Char('q') | KeyCode::Char('c') if ctrl => Some(Message::Quit),
        KeyCode::Char('y') => Some(Message::Yes),
        _ => None,
    }
}

fn update(model: &mut Model, msg: Message) -> Option<Message> {
    match msg {
        Message::Boot => {
            model.running_state = RunningState::Running;
        }
        Message::Quit => {
            model.running_state = RunningState::Done;
        }
        _ => {}
    };
    None
}

mod tui {
    use ratatui::{
        DefaultTerminal, Terminal,
        backend::CrosstermBackend,
        crossterm::{
            ExecutableCommand,
            terminal::{
                EnterAlternateScreen, LeaveAlternateScreen, disable_raw_mode, enable_raw_mode,
            },
        },
    };
    use std::{io::stdout, panic};

    pub fn init_terminal() -> color_eyre::Result<DefaultTerminal> {
        enable_raw_mode()?;
        stdout().execute(EnterAlternateScreen)?;
        let terminal = Terminal::new(CrosstermBackend::new(stdout()))?;
        Ok(terminal)
    }

    pub fn restore_terminal() -> color_eyre::Result<()> {
        stdout().execute(LeaveAlternateScreen)?;
        disable_raw_mode()?;
        Ok(())
    }

    pub fn install_panic_hook() {
        let original_hook = panic::take_hook();
        panic::set_hook(Box::new(move |panic_info| {
            stdout().execute(LeaveAlternateScreen).unwrap();
            disable_raw_mode().unwrap();
            original_hook(panic_info);
        }));
    }
}
