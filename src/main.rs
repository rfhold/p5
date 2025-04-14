use clap::Parser;
use color_eyre::Result;
use contexts::{Action, AppContext, AppContextKey, ProgramAction};
use model::Model;
use p5::{ContextAction, ContextController};
use ui::layout::Layout;

mod contexts;
mod model;
mod pulumi;
mod ui;

#[derive(Parser)]
#[command(name = "p5")]
#[command(bin_name = "p5")]
struct P5 {}

#[tokio::main]
async fn main() -> Result<()> {
    let _args = P5::parse();

    tui::install_panic_hook();
    let terminal = tui::init_terminal()?;
    let model = Model {
        context_order: vec![
            AppContextKey::Programs,
            AppContextKey::Stacks,
            AppContextKey::OperationView,
            AppContextKey::Status,
        ],
        ..Default::default()
    };

    let controller = ContextController::new(model).with_boot_actions(vec![
        ContextAction::AppAction(Action::ProgramAction(ProgramAction::List)),
        ContextAction::AppAction(Action::SetContext(AppContextKey::Programs)),
    ]);

    controller
        .run(terminal, AppContext::default(), Layout::new())
        .await?;

    tui::restore_terminal()?;
    Ok(())
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
