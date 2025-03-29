use std::io::stdout;
use std::time::Duration;
use std::{ffi::OsString, process::Command};

use color_eyre::Result;
use crossterm::ExecutableCommand;
use crossterm::event::{self, Event, KeyCode, KeyEvent, KeyEventKind};
use crossterm::terminal::{
    EnterAlternateScreen, LeaveAlternateScreen, disable_raw_mode, enable_raw_mode,
};
use layout::Flex;
use ratatui::prelude::*;
use ratatui::style::palette::tailwind::{BLUE, GREEN, SLATE};
use ratatui::style::{Color, Modifier, Style};
use ratatui::text::Line;
use ratatui::widgets::{
    Block, Borders, HighlightSpacing, List, ListDirection, ListItem, ListState, Padding, Paragraph,
    StatefulWidget, Tabs, Widget, Wrap,
};
use ratatui::{DefaultTerminal, Frame, symbols};

use strum::{Display, EnumIter, FromRepr, IntoEnumIterator};

use clap::{Parser, arg, command};

mod auto;
mod pulumi;

use pulumi::{ProgramBackend, Provider};

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
    let args = P5::parse();

    tui::install_panic_hook();
    let mut terminal = tui::init_terminal()?;
    let mut model = Model::new(args.backend, args.secrets_provider);

    while model.running_state != RunningState::Done {
        // Render the current view
        terminal.draw(|f| view(&mut model, f))?;

        // Handle events and map to a Message
        let mut current_msg = handle_event(&model)?;

        // Process updates as long as they return a non-None message
        while current_msg.is_some() {
            current_msg = update(&mut model, current_msg.unwrap(), &mut terminal);
        }
    }

    tui::restore_terminal()?;
    Ok(())
}

struct Model {
    running_state: RunningState,
    left_vertical_panes: Vec<Pane>,
    left_vertical_panes_state: ListState,
    main_tab_panes: Vec<Pane>,
    main_tab_selected: Option<Pane>,
    program_list: ProgramList,
    stack_list: StackList,
    resources: ResourceTab,
    preview: PreviewTab,
    selected_stack: Option<String>,
    pulumi_backend: pulumi::LocalProgramBackend,
    secrets_provider: Option<String>,
    error_popup: Option<String>,
    delete_confirmation: Option<(String, Option<String>)>,
    import_resource: Option<ResourceImport>,
    selected_provider: Option<Provider>,
}

impl Model {
    fn new(backend: Option<String>, secrets_provider: Option<String>) -> Self {
        Self {
            running_state: RunningState::default(),
            left_vertical_panes: vec![Pane::ProgramList, Pane::StackList],
            left_vertical_panes_state: ListState::default(),
            main_tab_panes: vec![Pane::Resources, Pane::Preview],
            main_tab_selected: None,
            program_list: ProgramList {
                loading_state: LoadingState::Waiting,
                items: vec![],
                state: ListState::default(),
            },
            stack_list: StackList {
                loading_state: LoadingState::Waiting,
                items: vec![],
                state: ListState::default(),
            },
            error_popup: None,
            resources: ResourceTab {
                state: LoadingState::Waiting,
                resource_list: ResourceList {
                    items: vec![],
                    state: ListState::default(),
                },
            },
            preview: PreviewTab {
                state: LoadingState::Waiting,
                preview_list: PreviewList {
                    items: vec![],
                    state: ListState::default(),
                },
            },
            selected_stack: None,
            secrets_provider,
            pulumi_backend: pulumi::LocalProgramBackend::new(pulumi::PulumiContext {
                cwd: ".".to_string(),
                backend_url: backend,
                env: vec![],
            }),
            delete_confirmation: Option::None,
            import_resource: Option::None,
            selected_provider: Option::None,
        }
    }
}

struct ProgramList {
    loading_state: LoadingState,
    items: Vec<ProgramItem>,
    state: ListState,
}

struct ProgramItem {
    name: String,
    path: String,
}

struct StackList {
    loading_state: LoadingState,
    items: Vec<StackItem>,
    state: ListState,
}

struct StackItem {
    name: String,
    initialized: bool,
    secret_provider: Option<String>,
}

#[derive(Display)]
enum Pane {
    #[strum(to_string = "Programs")]
    ProgramList,
    #[strum(to_string = "Stacks")]
    StackList,
    #[strum(to_string = "Config")]
    StackConfig,
    #[strum(to_string = "Resources")]
    Resources,
    #[strum(to_string = "Preview")]
    Preview,
    #[strum(to_string = "Init Stack")]
    InitConfirmation,
    #[strum(to_string = "Import Resource")]
    ResourceImport,
}

struct ResourceImport {
    stack: String,
    resource_type: String,
    name: String,
    id: String,
    options: Option<Vec<String>>,
    options_state: ListState,
}

struct InitConfirmation {
    stack: StackItem,
}

struct StackConfig {
    state: LoadingState,
    stack: StackItem,
    config: String,
}

struct ResourceTab {
    state: LoadingState,
    resource_list: ResourceList,
}

struct ResourceList {
    items: Vec<ResourceItem>,
    state: ListState,
}

struct ResourceItem {
    urn: String,
    name: String,
    type_: Option<String>,
}

struct PreviewTab {
    state: LoadingState,
    preview_list: PreviewList,
}

struct PreviewList {
    items: Vec<PreviewItem>,
    state: ListState,
}

#[derive(Clone)]
struct PreviewItem {
    urn: String,
    chage_type: String,
}

#[derive(Default, Clone, Copy, Display, FromRepr, EnumIter)]
enum ChangeType {
    #[default]
    #[strum(to_string = "Create")]
    Create,
    #[strum(to_string = "Update")]
    Update,
    #[strum(to_string = "Delete")]
    Delete,
}

#[derive(Debug, Default, PartialEq, Eq)]
enum RunningState {
    #[default]
    Boot,
    Running,
    Done,
}

#[derive(Debug, Default, PartialEq, Eq)]
enum LoadingState {
    #[default]
    Waiting,
    Loading,
    Error,
    Done,
}

#[derive(PartialEq)]
enum Message {
    Yes,
    No,
    Boot,
    Back,
    Reset,
    Reload,
    Quit,
    NavUp,
    NavDown,
    NavLeft,
    NavRight,
    RemoveChar,
    AddChar(char),
    CommandPrompt,
    Command(String),
    Toggle,
    SetCwd(String),
    SelectProgram(String),
    SelectStack(String),
    Edit,
    ConfigStack(String),
    Config,
    MarkProvider,
    ShowStackConfig(String),
    Preview,
    PreviewStack(String),
    List,
    ListStackResources(String),
    Import,
    ImportPrompt(String, String, String, String, Option<Vec<String>>),
    ImportResource(String, String, String, String),
    New,
    InitStack(String, String),
    Refresh,
    RefreshStack(String),
    RefreshResource(String, String),
    Destroy,
    DestroyStack(String),
    DestroyResource(String, String),
    Delete,
    DeleteStack(String),
    DeleteResource(String, String),
    Update,
    UpdateStack(String),
    UpdateResource(String, String),
    Rename,
    RenameStack(String, String),
    RenameResource(String, String, String),
    ScanStackFiles(String),
    StartFetchPrograms,
    FetchPrograms,
    StartFetchStacks,
    FetchStacks,
    DeleteConfirmation(String, Option<String>),
    StartFetchResources(String),
    StartFetchPreview(String),
    StartFetchConfig(String),
    Confirm,
    AutoImport,
    AutoImportResource(String, String),
}

fn view(model: &mut Model, frame: &mut Frame) {
    let main_layout = Layout::default()
        .direction(Direction::Horizontal)
        .constraints(vec![Constraint::Percentage(30), Constraint::Percentage(70)])
        .split(frame.area());

    let side_layout = Layout::default()
        .direction(Direction::Vertical)
        .constraints(vec![Constraint::Percentage(50), Constraint::Percentage(50)])
        .split(main_layout[0]);

    let items = model
        .program_list
        .items
        .iter()
        .map(|item| ListItem::new(item.name.clone()))
        .collect::<Vec<_>>();

    let program_list = List::new(items)
        .block(Block::default().borders(Borders::ALL).title(format!(
            "Programs: ({:?})",
            model.program_list.loading_state
        )))
        .highlight_style(
            Style::default()
                .fg(Color::LightBlue)
                .add_modifier(Modifier::BOLD),
        )
        .highlight_symbol(">")
        .style(
            Style::default().fg(if model.left_vertical_panes_state.selected() == Some(0) {
                Color::White
            } else {
                Color::DarkGray
            }),
        )
        .direction(ListDirection::TopToBottom);

    frame.render_stateful_widget(program_list, side_layout[0], &mut model.program_list.state);

    let items = model
        .stack_list
        .items
        .iter()
        .map(|item| {
            ListItem::new(item.name.clone()).style(match item.initialized {
                true => Style::default(),
                false => Style::default().fg(Color::DarkGray),
            })
        })
        .collect::<Vec<_>>();

    let stack_list = List::new(items)
        .block(
            Block::default()
                .borders(Borders::ALL)
                .title(format!("Stacks: ({:?})", model.stack_list.loading_state)),
        )
        .highlight_style(
            Style::default()
                .fg(Color::LightBlue)
                .add_modifier(Modifier::BOLD),
        )
        .highlight_symbol(">")
        .style(
            Style::default().fg(if model.left_vertical_panes_state.selected() == Some(1) {
                Color::White
            } else {
                Color::DarkGray
            }),
        )
        .direction(ListDirection::TopToBottom);

    frame.render_stateful_widget(stack_list, side_layout[1], &mut model.stack_list.state);

    match model.main_tab_selected {
        Some(Pane::Resources) | None => {
            let items = model
                .resources
                .resource_list
                .items
                .iter()
                .map(|item| {
                    ListItem::new(item.urn.clone()).style(Style::default().bg(
                        match model.selected_provider.clone() {
                            Some(provider) if provider.urn == item.urn => Color::Green,
                            _ => Color::Black,
                        },
                    ))
                })
                .collect::<Vec<_>>();

            let resource_list = List::new(items)
                .block(
                    Block::default()
                        .borders(Borders::ALL)
                        .title(format!("Resources: ({:?})", model.resources.state)),
                )
                .highlight_style(
                    Style::default()
                        .fg(Color::LightBlue)
                        .add_modifier(Modifier::BOLD),
                )
                .highlight_symbol(">")
                .style(Style::default().fg(if model.selected_stack.is_some() {
                    Color::White
                } else {
                    Color::DarkGray
                }))
                .direction(ListDirection::TopToBottom);

            frame.render_stateful_widget(
                resource_list,
                main_layout[1],
                &mut model.resources.resource_list.state,
            );
        }
        Some(Pane::Preview) | _ => {
            let items = model
                .preview
                .preview_list
                .items
                .iter()
                .map(|item| {
                    ListItem::new(
                        format!("{:?} {:?}", item.chage_type, item.urn)
                            .bg(match item.chage_type.as_str() {
                                "read" => Color::Blue,
                                "create" => Color::Green,
                                "delete" => Color::Red,
                                "update" => Color::Yellow,
                                _ => Color::White,
                            })
                            .fg(Color::Black),
                    )
                })
                .collect::<Vec<_>>();

            let preview_list = List::new(items)
                .block(
                    Block::default()
                        .borders(Borders::ALL)
                        .title(format!("Preview: ({:?})", model.preview.state)),
                )
                .highlight_style(Style::default().add_modifier(Modifier::BOLD))
                .highlight_symbol(">")
                .style(Style::default().fg(if model.selected_stack.is_some() {
                    Color::White
                } else {
                    Color::DarkGray
                }))
                .direction(ListDirection::TopToBottom);

            frame.render_stateful_widget(
                preview_list,
                main_layout[1],
                &mut model.preview.preview_list.state,
            );
        }
    }

    if let Some(error) = &model.error_popup {
        let area = popup_area(frame.area(), 50, 50);
        let popup = Paragraph::new(error.clone())
            .on_black()
            .block(
                Block::default()
                    .on_black()
                    .borders(Borders::ALL)
                    .style(Style::default().bg(Color::Red).fg(Color::White))
                    .title("Error")
                    .title_style(Style::default().fg(Color::Red).add_modifier(Modifier::BOLD)),
            )
            .wrap(Wrap { trim: true });

        frame.render_widget(popup, area);
    }

    if let Some((stack, resource)) = &model.delete_confirmation {
        let message = match resource {
            Some(resource) => format!("Delete resource {} from stack {}?", resource, stack),
            None => format!("Delete stack {}?", stack),
        };

        let area = popup_area(frame.area(), 50, 50);

        let confirmation_layout = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([Constraint::Percentage(80), Constraint::Percentage(20)])
            .split(area);

        let controls = " [y]es/[n]o ";

        let popup = Paragraph::new(message)
            .on_black()
            .block(
                Block::default()
                    .on_black()
                    .borders(Borders::ALL)
                    .style(Style::default().bg(Color::Red).fg(Color::White))
                    .title("Delete Confirmation")
                    .title_style(Style::default().fg(Color::Red).add_modifier(Modifier::BOLD)),
            )
            .wrap(Wrap { trim: true });

        frame.render_widget(popup, confirmation_layout[0]);

        let controls = Paragraph::new(controls)
            .on_black()
            .block(
                Block::default()
                    .on_black()
                    .borders(Borders::ALL)
                    .style(Style::default().bg(Color::Red).fg(Color::White))
                    .title("Controls")
                    .title_style(Style::default().fg(Color::Red).add_modifier(Modifier::BOLD)),
            )
            .wrap(Wrap { trim: true });

        frame.render_widget(controls, confirmation_layout[1]);
    }

    // show import prompt with id text input
    if let Some(import) = &mut model.import_resource {
        let area = popup_area(frame.area(), 50, 50);

        if let Some(options) = &import.options {
            let items = options
                .iter()
                .map(|item| ListItem::new(item.clone()))
                .collect::<Vec<_>>();

            let options_list = List::new(items)
                .on_black()
                .block(
                    Block::default()
                        .on_black()
                        .borders(Borders::ALL)
                        .title("Options")
                        .title_style(
                            Style::default()
                                .fg(Color::Green)
                                .add_modifier(Modifier::BOLD),
                        ),
                )
                .highlight_style(
                    Style::default()
                        .fg(Color::LightBlue)
                        .add_modifier(Modifier::BOLD),
                )
                .highlight_symbol(">")
                .style(Style::default().fg(Color::White))
                .direction(ListDirection::TopToBottom);

            frame.render_stateful_widget(options_list, area, &mut import.options_state);
        } else {
            let layout = Layout::default()
                .direction(Direction::Vertical)
                .constraints([
                    Constraint::Percentage(60),
                    Constraint::Percentage(20),
                    Constraint::Percentage(20),
                ])
                .split(area);

            let message = format!(
                "Import resource {} {} {} into stack {}?",
                import.resource_type, import.name, import.id, import.stack
            );

            let popup = Paragraph::new(message)
                .on_black()
                .block(
                    Block::default()
                        .on_black()
                        .borders(Borders::ALL)
                        .style(Style::default().bg(Color::Black).fg(Color::White))
                        .title("Import Resource")
                        .title_style(
                            Style::default()
                                .fg(Color::Green)
                                .add_modifier(Modifier::BOLD),
                        ),
                )
                .wrap(Wrap { trim: true });

            frame.render_widget(popup, layout[0]);

            let provider_text = match &model.selected_provider {
                Some(provider) => format!("Provider: {}={}", provider.name, provider.urn),
                None => "No provider selected".to_string(),
            };

            let provider = Paragraph::new(provider_text)
                .on_black()
                .block(
                    Block::default()
                        .on_black()
                        .borders(Borders::ALL)
                        .style(Style::default().bg(Color::Black).fg(Color::White))
                        .title("Provider")
                        .title_style(
                            Style::default()
                                .fg(Color::Green)
                                .add_modifier(Modifier::BOLD),
                        ),
                )
                .wrap(Wrap { trim: true });

            frame.render_widget(provider, layout[1]);

            let input = Paragraph::new(import.id.clone())
                .on_black()
                .block(
                    Block::default()
                        .on_black()
                        .borders(Borders::ALL)
                        .style(Style::default().bg(Color::Black).fg(Color::White))
                        .title("ID")
                        .title_style(
                            Style::default()
                                .fg(Color::Green)
                                .add_modifier(Modifier::BOLD),
                        ),
                )
                .wrap(Wrap { trim: true });

            frame.render_widget(input, layout[2]);

            frame.set_cursor_position(Position {
                x: layout[2].x + 1 + import.id.len() as u16,
                y: layout[2].y + 1,
            });
        }
    }
}

fn popup_area(area: Rect, percent_x: u16, percent_y: u16) -> Rect {
    let vertical = Layout::vertical([Constraint::Percentage(percent_y)]).flex(Flex::Center);
    let horizontal = Layout::horizontal([Constraint::Percentage(percent_x)]).flex(Flex::Center);
    let [area] = vertical.areas(area);
    let [area] = horizontal.areas(area);
    area
}

fn handle_event(model: &Model) -> color_eyre::Result<Option<Message>> {
    if model.running_state == RunningState::Boot {
        return Ok(Some(Message::Boot));
    }

    if event::poll(Duration::from_millis(250))? {
        if let Event::Key(key) = event::read()? {
            if key.kind == event::KeyEventKind::Press {
                return Ok(handle_key(key, model));
            }
        }
    }
    Ok(None)
}

fn handle_key(key: event::KeyEvent, model: &Model) -> Option<Message> {
    if let Some(import) = &model.import_resource {
        if let None = import.options {
            return match key.code {
                KeyCode::Esc => Some(Message::Back),
                KeyCode::Enter => Some(Message::Toggle),
                KeyCode::Backspace => Some(Message::RemoveChar),
                KeyCode::Char(c) => Some(Message::AddChar(c)),
                _ => None,
            };
        }
    }

    let shift = key.modifiers.contains(event::KeyModifiers::SHIFT);
    let ctrl = key.modifiers.contains(event::KeyModifiers::CONTROL);
    match key.code {
        KeyCode::Char('a') => Some(Message::AutoImport),
        KeyCode::Char('q') => Some(Message::Quit),
        KeyCode::Char('j') | KeyCode::Down => Some(Message::NavDown),
        KeyCode::Char('k') | KeyCode::Up => Some(Message::NavUp),
        KeyCode::Char('h') | KeyCode::Left => Some(Message::NavLeft),
        KeyCode::Char('l') | KeyCode::Right => Some(Message::NavRight),
        KeyCode::Char('c') if ctrl => Some(Message::Quit),
        KeyCode::Char('c') => Some(Message::Config),
        KeyCode::Char('e') => Some(Message::Edit),
        KeyCode::Char('p') if ctrl => Some(Message::MarkProvider),
        KeyCode::Char('P') if shift => Some(Message::Preview),
        KeyCode::Char('d') if ctrl => Some(Message::Delete),
        KeyCode::Char('D') if shift => Some(Message::Destroy),
        KeyCode::Char('i') => Some(Message::Import),
        KeyCode::Char('n') => Some(Message::New),
        KeyCode::Char('U') if shift => Some(Message::Update),
        KeyCode::Char('x') => Some(Message::Delete),
        KeyCode::Char('y') => Some(Message::Yes),
        KeyCode::Char('R') if shift => Some(Message::Reload),
        KeyCode::Char('r') if ctrl => Some(Message::Reset),
        KeyCode::Char('r') => Some(Message::Refresh),
        KeyCode::Char(':') => Some(Message::CommandPrompt),
        KeyCode::Char(' ') | KeyCode::Enter => Some(Message::Toggle),
        KeyCode::Esc | KeyCode::Backspace => Some(Message::Back),
        _ => None,
    }
}

fn update(model: &mut Model, msg: Message, terminal: &mut DefaultTerminal) -> Option<Message> {
    match msg {
        Message::Boot => {
            model.running_state = RunningState::Running;
            return Some(Message::StartFetchPrograms);
        }
        Message::Back => {
            if let Some(_) = model.import_resource {
                model.import_resource = None;
            } else if let Some(_) = model.delete_confirmation {
                model.delete_confirmation = None;
            } else if let Some(_) = model.error_popup {
                model.error_popup = None;
            } else if let Some(_) = &model.selected_stack {
                model.main_tab_selected = None;
                model.selected_stack = None;
                model.left_vertical_panes_state.select(Some(1));
            } else if let Some(_) = model.left_vertical_panes_state.selected() {
                model.left_vertical_panes_state.select_previous();
            } else {
                return Some(Message::Quit);
            }
        }
        Message::RemoveChar => {
            if let Some(import) = &mut model.import_resource {
                import.id.pop();
            }
        }
        Message::AddChar(c) => {
            if let Some(import) = &mut model.import_resource {
                import.id.push(c);
            }
        }
        Message::Quit => {
            model.running_state = RunningState::Done;
        }
        Message::StartFetchPrograms => {
            model.program_list.loading_state = LoadingState::Loading;
            return Some(Message::FetchPrograms);
        }
        Message::FetchPrograms => match pulumi::find_programs(3) {
            Ok(programs) => {
                model.program_list.items = programs
                    .into_iter()
                    .map(|program| ProgramItem {
                        name: program.config.name.clone(),
                        path: program.path,
                    })
                    .collect();
                model.program_list.loading_state = LoadingState::Done;
                model.left_vertical_panes_state.select(Some(0));
                model.program_list.state.select(Some(0));
            }
            Err(error) => {
                model.error_popup = Some(format!("Error fetching programs: {:?}", error));
                model.program_list.loading_state = LoadingState::Error;
            }
        },
        Message::SelectProgram(program_path) => {
            model.pulumi_backend.set_cwd(program_path);
            model.selected_stack = None;

            return Some(Message::StartFetchStacks);
        }
        Message::StartFetchStacks => {
            model.stack_list.loading_state = LoadingState::Loading;
            return Some(Message::FetchStacks);
        }
        Message::Update if model.selected_stack.is_some() => match model.main_tab_selected {
            Some(Pane::Preview) => {
                return Some(Message::UpdateStack(model.selected_stack.clone().unwrap()));
            }
            _ => {}
        },
        Message::ConfigStack(_stack) => {
            run_editor(terminal, &format!("{}", model.pulumi_backend.cwd()));
        }
        Message::UpdateStack(stack) => match model.pulumi_backend.update_stack(&stack) {
            Ok(_) => {
                return Some(Message::Reload);
            }
            Err(error) => {
                model.error_popup = Some(format!("Error updating stack: {:?}", error));
            }
        },
        Message::FetchStacks => match model.pulumi_backend.list_stacks() {
            Ok(stacks) => {
                model.stack_list.items = stacks
                    .into_iter()
                    .map(|stack| StackItem {
                        name: stack.name,
                        initialized: true,
                        secret_provider: None,
                    })
                    .collect();
                model.stack_list.loading_state = LoadingState::Done;
                model.left_vertical_panes_state.select(Some(1));
                model.stack_list.state.select(Some(0));

                return Some(Message::ScanStackFiles(
                    model.pulumi_backend.cwd().to_string(),
                ));
            }
            Err(error) => {
                model.error_popup = Some(format!("Error fetching stacks: {:?}", error));
                model.stack_list.loading_state = LoadingState::Error;
            }
        },
        Message::ScanStackFiles(cwd) => match pulumi::find_stack_files(&cwd) {
            Ok(stacks) => {
                for stack in stacks {
                    let stack_name = stack.to_string();
                    if !model
                        .stack_list
                        .items
                        .iter()
                        .any(|stack| stack.name == stack_name)
                    {
                        model.stack_list.items.push(StackItem {
                            name: stack_name,
                            initialized: false,
                            secret_provider: None,
                        });
                    }
                }
            }
            Err(error) => {
                model.error_popup = Some(format!("Error scanning stack files: {:?}", error));
            }
        },
        Message::SelectStack(stack) => {
            model.selected_stack = Some(stack.clone());
            model.left_vertical_panes_state.select(None);

            return Some(Message::StartFetchResources(stack));
        }
        Message::ShowStackConfig(stack) => todo!(),
        Message::PreviewStack(stack) => match model.pulumi_backend.preview_stack(&stack) {
            Ok(preview) => {
                model.preview.preview_list.items = preview
                    .steps
                    .into_iter()
                    .filter(|step| step.op != "same")
                    .map(|resource| PreviewItem {
                        urn: resource.urn.clone(),
                        chage_type: resource.op,
                    })
                    .collect();
                model.preview.state = LoadingState::Done;
                model.preview.preview_list.state.select(Some(0));
                model.main_tab_selected = Some(Pane::Preview);
            }
            Err(error) => {
                model.error_popup = Some(format!("Error fetching preview: {:?}", error));
                model.preview.state = LoadingState::Error;
            }
        },
        Message::StartFetchPreview(stack) => {
            model.preview.state = LoadingState::Loading;
            return Some(Message::PreviewStack(stack));
        }
        Message::StartFetchResources(stack) => {
            model.resources.state = LoadingState::Loading;
            return Some(Message::ListStackResources(stack));
        }
        Message::ListStackResources(stack) => {
            match model.pulumi_backend.list_stack_resources(&stack) {
                Ok(resources) => {
                    model.resources.resource_list.items = resources
                        .into_iter()
                        .map(|resource| ResourceItem {
                            urn: resource.urn.clone(),
                            // name or urn if name is None
                            name: resource
                                .name
                                .or(Some(resource.urn.split("::").last().unwrap().to_string()))
                                .unwrap(),
                            type_: resource.type_,
                        })
                        .collect();
                    model.resources.state = LoadingState::Done;
                    model.resources.resource_list.state.select(Some(0));
                    model.main_tab_selected = Some(Pane::Resources);
                }
                Err(error) => {
                    model.error_popup = Some(format!("Error fetching resources: {:?}", error));
                    model.resources.state = LoadingState::Error;
                }
            }
        }
        Message::AutoImport => {
            if let Some(selected_stack) = &model.selected_stack {
                if let Some(tab) = &model.main_tab_selected {
                    if let Some(index) = model.preview.preview_list.state.selected() {
                        let preview = model.preview.preview_list.items[index].clone();
                        return Some(Message::AutoImportResource(
                            selected_stack.clone(),
                            preview.urn,
                        ));
                    }
                }
            }
        }
        Message::AutoImportResource(stack, urn) => match auto::guess_id_for_urn(&urn) {
            Ok((resource_type, name, id, options)) => {
                return Some(Message::ImportPrompt(
                    stack,
                    resource_type,
                    name,
                    id,
                    options,
                ));
            }
            Err(error) => {
                model.error_popup = Some(format!("Error auto-importing resource: {:?}", error));
            }
        },
        Message::ImportPrompt(stack, resource_type, name, id, options) => {
            model.import_resource = Some(ResourceImport {
                stack,
                resource_type,
                name,
                id,
                options,
                options_state: ListState::default(),
            });
        }
        Message::ImportResource(stack, type_, name, id) => {
            match model.pulumi_backend.import_stack_resource(
                &stack,
                &type_,
                &name,
                &id,
                model.selected_provider.clone(),
            ) {
                Ok(_) => {
                    model.import_resource = None;
                    return Some(Message::Reload);
                }
                Err(error) => {
                    model.error_popup = Some(format!("Error importing resource: {:?}", error));
                }
            }
        }

        Message::InitStack(stack, secrets_provider) => {
            match model.pulumi_backend.init_stack(&stack, &secrets_provider) {
                Ok(_) => {
                    return Some(Message::StartFetchStacks);
                }
                Err(error) => {
                    model.error_popup = Some(format!("Error initializing stack: {:?}", error));
                }
            }
        }
        Message::Refresh => match &model.selected_stack {
            Some(stack) => {
                return Some(Message::RefreshStack(stack.clone()));
            }
            None => match model.left_vertical_panes_state.selected() {
                Some(1) => {
                    if let Some(index) = model.stack_list.state.selected() {
                        return Some(Message::RefreshStack(
                            model.stack_list.items[index].name.clone(),
                        ));
                    }
                }
                _ => {}
            },
        },
        Message::RefreshStack(stack) => match model.pulumi_backend.refresh_stack(&stack) {
            Ok(_) => {
                return Some(Message::Reload);
            }
            Err(error) => {
                model.error_popup = Some(format!("Error refreshing stack: {:?}", error));
            }
        },
        Message::RefreshResource(stack, urn) => todo!(),
        Message::DeleteStack(stack) => match model.pulumi_backend.delete_stack(&stack) {
            Ok(_) => {
                return Some(Message::Reload);
            }
            Err(error) => {
                model.error_popup = Some(format!("Error deleting stack: {:?}", error));
            }
        },
        Message::MarkProvider => {
            if let Some(selected_stack) = &model.selected_stack {
                model.selected_provider = match model.main_tab_selected {
                    Some(Pane::Resources) => {
                        if let Some(index) = model.resources.resource_list.state.selected() {
                            let item = &model.resources.resource_list.items[index];
                            Some(Provider {
                                name: item.name.clone(),
                                urn: item.urn.clone(),
                            })
                        } else {
                            None
                        }
                    }
                    Some(Pane::Preview) => {
                        if let Some(index) = model.preview.preview_list.state.selected() {
                            let item = &model.preview.preview_list.items[index];
                            Some(Provider {
                                name: item.urn.clone(),
                                urn: item.urn.clone(),
                            })
                        } else {
                            None
                        }
                    }
                    _ => None,
                };

                if let Some(Pane::Preview) = model.main_tab_selected {
                    return Some(Message::AutoImport);
                }
            }
        }
        Message::DeleteResource(stack, urn) => {
            match model.pulumi_backend.delete_stack_resource(&stack, &urn) {
                Ok(_) => {
                    model.delete_confirmation = None;
                    return Some(Message::Reload);
                }
                Err(error) => {
                    model.error_popup = Some(format!("Error deleting resource: {:?}", error));
                }
            }
        }
        Message::Config => todo!(),
        Message::Preview => {
            if let Some(selected_stack) = &model.selected_stack {
                model.main_tab_selected = Some(Pane::Preview);
                return Some(Message::StartFetchPreview(selected_stack.clone()));
            }
        }
        Message::List => todo!(),
        Message::Import => todo!(),
        Message::New if model.delete_confirmation.is_some() => {
            return Some(Message::No);
        }
        Message::New => match model.selected_stack {
            None => {
                if let Some(selected) = model.left_vertical_panes_state.selected() {
                    match model.left_vertical_panes[selected] {
                        Pane::StackList => {
                            if let Some(index) = model.stack_list.state.selected() {
                                if model.stack_list.items[index].initialized {
                                    model.error_popup =
                                        Some("Stack already initialized".to_string());
                                } else {
                                    return Some(Message::InitStack(
                                        model.stack_list.items[index].name.clone(),
                                        model.secrets_provider.clone().unwrap(),
                                    ));
                                }
                            }
                        }
                        _ => {}
                    }
                }
            }
            Some(_) => {}
        },
        Message::Reload => match model.main_tab_selected {
            Some(Pane::Resources) => {
                if let Some(selected_stack) = &model.selected_stack {
                    return Some(Message::StartFetchResources(selected_stack.clone()));
                }
            }
            Some(Pane::Preview) => {
                if let Some(selected_stack) = &model.selected_stack {
                    return Some(Message::StartFetchPreview(selected_stack.clone()));
                }
            }
            _ => match model.left_vertical_panes_state.selected() {
                Some(0) => {
                    return Some(Message::StartFetchPrograms);
                }
                Some(1) => {
                    return Some(Message::StartFetchStacks);
                }
                _ => {}
            },
        },
        Message::Update => todo!(),
        Message::Delete => match &model.selected_stack {
            None => {
                if let Some(selected) = model.left_vertical_panes_state.selected() {
                    match model.left_vertical_panes[selected] {
                        Pane::StackList => {
                            if let Some(index) = model.stack_list.state.selected() {
                                let stack = model.stack_list.items[index].name.clone();
                                model.delete_confirmation = Some((stack, None));
                            }
                        }
                        _ => {}
                    }
                }
            }
            Some(stack) => match model.main_tab_selected {
                Some(Pane::Resources) => {
                    if let Some(index) = model.resources.resource_list.state.selected() {
                        let resource = model.resources.resource_list.items[index].urn.clone();
                        model.delete_confirmation = Some((stack.clone(), Some(resource)));
                    }
                }
                _ => {}
            },
        },
        Message::Yes if model.delete_confirmation.is_some() => {
            let conf = model.delete_confirmation.clone();
            model.delete_confirmation = None;
            match conf {
                Some((stack, Some(resource))) => {
                    return Some(Message::DeleteResource(stack.clone(), resource.clone()));
                }
                Some((stack, None)) => {
                    return Some(Message::DeleteStack(stack.clone()));
                }
                _ => {}
            }
        }
        Message::No if model.delete_confirmation.is_some() => {
            model.delete_confirmation = None;
        }
        Message::NavUp | Message::NavDown
            if model.delete_confirmation.is_some() || model.error_popup.is_some() => {}
        Message::NavUp | Message::NavDown if model.import_resource.is_some() => match model
            .import_resource
            .as_mut()
            .map(|import| &mut import.options_state)
        {
            Some(state) => match msg {
                Message::NavUp => state.select_previous(),
                Message::NavDown => state.select_next(),
                _ => {}
            },
            None => {}
        },
        Message::Edit => match &model.selected_stack {
            Some(stack) => {
                return Some(Message::ConfigStack(stack.clone()));
            }
            None => {}
        },
        Message::Toggle => {
            if let Some(import) = &model.import_resource {
                return match &import.options {
                    Some(opts) => {
                        let selected = import.options_state.selected();
                        match selected {
                            Some(index) => Some(Message::ImportPrompt(
                                import.stack.clone(),
                                import.resource_type.clone(),
                                import.name.clone(),
                                opts[index].clone(),
                                None,
                            )),
                            None => None,
                        }
                    }
                    None => Some(Message::ImportResource(
                        import.stack.clone(),
                        import.resource_type.clone(),
                        import.name.clone(),
                        import.id.clone(),
                    )),
                };
            }
            if let Some(selected) = model.left_vertical_panes_state.selected() {
                match model.left_vertical_panes[selected] {
                    Pane::ProgramList => {
                        if let Some(index) = model.program_list.state.selected() {
                            return Some(Message::SelectProgram(
                                model.program_list.items[index].path.clone(),
                            ));
                        }
                    }
                    Pane::StackList => {
                        if let Some(index) = model.stack_list.state.selected() {
                            return Some(Message::SelectStack(
                                model.stack_list.items[index].name.clone(),
                            ));
                        }
                    }
                    _ => {}
                }
            }
        }
        Message::NavUp => {
            if let Some(selected) = model.left_vertical_panes_state.selected() {
                match model.left_vertical_panes[selected] {
                    Pane::ProgramList => {
                        model.program_list.state.select_previous();
                    }
                    Pane::StackList => {
                        model.stack_list.state.select_previous();
                    }
                    _ => {}
                }
            }
            if let Some(selected_stack) = &model.selected_stack {
                if let Some(tab) = &model.main_tab_selected {
                    match tab {
                        Pane::Resources => {
                            model.resources.resource_list.state.select_previous();
                        }
                        Pane::Preview => {
                            model.preview.preview_list.state.select_previous();
                        }
                        _ => {}
                    }
                }
            }
        }
        Message::NavDown => {
            if let Some(selected) = model.left_vertical_panes_state.selected() {
                match model.left_vertical_panes[selected] {
                    Pane::ProgramList => {
                        model.program_list.state.select_next();
                    }
                    Pane::StackList => {
                        model.stack_list.state.select_next();
                    }
                    _ => {}
                }
            }
            if let Some(selected_stack) = &model.selected_stack {
                if let Some(tab) = &model.main_tab_selected {
                    match tab {
                        Pane::Resources => {
                            model.resources.resource_list.state.select_next();
                        }
                        Pane::Preview => {
                            model.preview.preview_list.state.select_next();
                        }
                        _ => {}
                    }
                }
            }
        }
        Message::NavLeft => {
            if let Some(selected_stack) = &model.selected_stack {
                if let Some(tab) = &model.main_tab_selected {
                    match tab {
                        Pane::Resources => {
                            model.main_tab_selected = None;
                            model.selected_stack = None;
                            model.stack_list.state.select(Some(0));
                            model.left_vertical_panes_state.select(Some(1));
                        }
                        Pane::Preview => {
                            model.main_tab_selected = Some(Pane::Resources);
                        }
                        _ => {}
                    }
                } else {
                    model.main_tab_selected = Some(Pane::Preview);
                }
            }
        }
        Message::NavRight => {
            if let Some(selected_stack) = &model.selected_stack {
                if let Some(tab) = &model.main_tab_selected {
                    match tab {
                        Pane::Resources => {
                            model.main_tab_selected = Some(Pane::Preview);
                        }
                        Pane::Preview => {
                            model.main_tab_selected = Some(Pane::Resources);
                        }
                        _ => {}
                    }
                } else {
                    model.main_tab_selected = Some(Pane::Resources);
                }
            }
        }
        _ => {}
    };
    None
}

mod tui {
    use ratatui::{
        DefaultTerminal, Terminal,
        backend::{Backend, CrosstermBackend},
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

fn run_editor(terminal: &mut DefaultTerminal, path: &str) -> color_eyre::Result<()> {
    stdout().execute(LeaveAlternateScreen)?;
    disable_raw_mode()?;
    let editor = std::env::var("EDITOR").unwrap_or("vim".to_string());
    Command::new(editor).arg(path).status()?;
    stdout().execute(EnterAlternateScreen)?;
    enable_raw_mode()?;
    terminal.clear()?;
    Ok(())
}
