pub mod controller;

pub type Result<T> = color_eyre::Result<T>;

use ratatui::{crossterm::ExecutableCommand, widgets::StatefulWidget};
use tracing_subscriber::prelude::*;

pub async fn run<H, W>(
    handler: H,
    state: H::State,
    init_actions: Vec<H::Action>,
    widget: W,
) -> color_eyre::Result<()>
where
    H: controller::Handler + Send + Sync + Clone + 'static,
    W: StatefulWidget<State = H::State> + Send + Sync + Clone + 'static,
{
    install_tracing();
    install_panic_hook()?;

    let cancel_token = tokio_util::sync::CancellationToken::new();

    let terminal = init_terminal()?;

    let controller =
        controller::Controller::new(handler, state, init_actions, cancel_token.clone());

    tokio::select! {
        _ = controller.run(terminal, widget) => {
            tracing::info!("Run task completed");
        },
        _ = tokio::signal::ctrl_c() => {
            tracing::info!("Ctrl-C received");
            cancel_token.cancel();
        },
        _ = cancel_token.cancelled() => {
            tracing::info!("Cancellation token triggered, shutting down...");
        },
    }

    tracing::debug!("Shutting down...");
    restore_terminal()?;

    Ok(())
}

fn init_terminal() -> color_eyre::Result<ratatui::DefaultTerminal> {
    ratatui::crossterm::terminal::enable_raw_mode()?;
    std::io::stdout().execute(ratatui::crossterm::terminal::EnterAlternateScreen)?;
    let terminal =
        ratatui::Terminal::new(ratatui::backend::CrosstermBackend::new(std::io::stdout()))?;
    Ok(terminal)
}

fn restore_terminal() -> color_eyre::Result<()> {
    std::io::stdout().execute(ratatui::crossterm::terminal::LeaveAlternateScreen)?;
    ratatui::crossterm::terminal::disable_raw_mode()?;
    Ok(())
}

fn install_panic_hook() -> color_eyre::Result<()> {
    color_eyre::config::HookBuilder::default()
        .panic_section("consider reporting the bug on github")
        .install()
}

fn install_tracing() {
    let registry = tracing_subscriber::registry()
        .with(tracing_error::ErrorLayer::default())
        .with(tracing_subscriber::EnvFilter::from_default_env())
        .with(
            tracing_subscriber::fmt::layer()
                .with_thread_ids(false)
                .with_thread_names(false)
                .with_file(true)
                .with_line_number(true)
                .with_target(false)
                .compact(),
        );

    registry.init();
}
