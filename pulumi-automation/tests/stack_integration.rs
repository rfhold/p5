use pulumi_automation::{
    event::{EngineEvent, EventType},
    local::LocalWorkspace,
    stack::{
        Operation, PulumiProcessListener, Stack, StackChangeSummary, StackDestroyOptions,
        StackPreviewOptions, StackRefreshOptions, StackUpOptions,
    },
    workspace::Workspace,
};

const PROGRAMS_PATH: &str = "tests/fixtures/programs";
const PULUMI_CONFIG_PASSPHRASE_FILE: &str = "tests/fixtures/programs/passphrase.txt";

fn prep() {
    let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR not set");

    let pulumi_backend_path = std::path::Path::new(&manifest_dir).join(PROGRAMS_PATH);

    if !pulumi_backend_path.exists() {
        panic!(
            "Pulumi backend path does not exist: {}",
            pulumi_backend_path.to_str().unwrap()
        );
    }

    let pulumi_backend_value = format!("file://{}", pulumi_backend_path.to_str().unwrap());
    println!("[TEST] Setting up environment variables for Pulumi Automation tests");
    println!("[TEST] PULUMI_BACKEND_URL: {}", pulumi_backend_value,);
    println!(
        "[TEST] PULUMI_CONFIG_PASSPHRASE_FILE: {}",
        std::path::Path::new(&manifest_dir)
            .join(PULUMI_CONFIG_PASSPHRASE_FILE)
            .to_str()
            .unwrap()
    );
    unsafe {
        std::env::set_var("PULUMI_BACKEND_URL", pulumi_backend_value);
        let pulumi_config_passphrase_file =
            std::path::Path::new(&manifest_dir).join(PULUMI_CONFIG_PASSPHRASE_FILE);
        std::env::set_var(
            "PULUMI_CONFIG_PASSPHRASE_FILE",
            pulumi_config_passphrase_file.to_str().unwrap(),
        );
    }
}

fn get_workspace_for_program(program: &str) -> LocalWorkspace {
    let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR not set");
    let program_path = std::path::Path::new(&manifest_dir)
        .join(PROGRAMS_PATH)
        .join(program);

    LocalWorkspace::new(program_path.to_str().unwrap().to_string())
}

#[tokio::test]
async fn test_preview() {
    prep();
    let workspace = get_workspace_for_program("replacement");
    let stack = workspace
        .select_stack("test")
        .expect("Failed to select stack");

    let result = stack.preview(StackPreviewOptions {
        ..Default::default()
    });

    assert!(result.is_ok(), "Stack preview failed: {:?}", result);

    let summary = result.expect("Failed to get stack preview summary");

    let mut matched_cases = 0;

    for step in summary.steps {
        match matched_cases {
            0 if step.op == Operation::Replace => {
                matched_cases += 1;
            }
            1 if step.op == Operation::Replace => {
                matched_cases += 1;
            }
            _ => {
                println!("Unexpected step in sequence {}: {:?}", matched_cases, step);
            }
        }
    }

    assert!(
        matched_cases == 2,
        "Expected all steps to match, but found {} matched cases",
        matched_cases
    );
}

#[tokio::test]
async fn test_preview_read() {
    prep();
    let workspace = get_workspace_for_program("replacement");
    let stack = workspace
        .select_stack("test")
        .expect("Failed to select stack");

    let result = stack.preview(StackPreviewOptions {
        show_reads: Some(true),
        ..Default::default()
    });

    assert!(result.is_ok(), "Stack preview failed: {:?}", result);

    let summary = result.expect("Failed to get stack preview summary");

    let mut matched_cases = 0;

    for step in summary.steps {
        match matched_cases {
            0 if step.op == Operation::Read => {
                matched_cases += 1;
            }
            1 if step.op == Operation::Replace => {
                matched_cases += 1;
            }
            2 if step.op == Operation::Replace => {
                matched_cases += 1;
            }
            _ => {
                println!("Unexpected step in sequence {}: {:?}", matched_cases, step);
            }
        }
    }

    assert!(
        matched_cases == 3,
        "Expected all steps to match, but found {} matched cases",
        matched_cases
    );
}

#[tokio::test]
async fn test_preview_replace() {
    prep();
    let workspace = get_workspace_for_program("replacement");
    let stack = workspace
        .select_stack("test")
        .expect("Failed to select stack");

    let result = stack.preview(StackPreviewOptions {
        show_reads: Some(true),
        show_replacement_steps: Some(true),
        ..Default::default()
    });

    assert!(result.is_ok(), "Stack preview failed: {:?}", result);

    let summary = result.expect("Failed to get stack preview summary");

    let mut matched_cases = 0;

    for step in summary.steps {
        match matched_cases {
            0 if step.op == Operation::Read => {
                matched_cases += 1;
            }
            1 if step.op == Operation::CreateReplacement => {
                matched_cases += 1;
            }
            2 if step.op == Operation::Replace => {
                matched_cases += 1;
            }
            3 if step.op == Operation::CreateReplacement => {
                matched_cases += 1;
            }
            4 if step.op == Operation::Replace => {
                matched_cases += 1;
            }
            5 if step.op == Operation::DeleteReplaced => {
                matched_cases += 1;
            }
            6 if step.op == Operation::DeleteReplaced => {
                matched_cases += 1;
            }
            _ => {
                println!("Unexpected step in sequence {}: {:?}", matched_cases, step);
            }
        }
    }

    assert!(
        matched_cases == 7,
        "Expected all steps to match, but found {} matched cases",
        matched_cases
    );
}

#[tokio::test]
async fn test_up() {
    prep();
    let workspace = get_workspace_for_program("replacement");
    let stack = workspace
        .select_stack("up")
        .expect("Failed to select stack");

    let (preview_tx, mut preview_rx) = tokio::sync::mpsc::channel::<StackChangeSummary>(100);
    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);

    let preview_listener = tokio::spawn(async move {
        while let Some(preview) = preview_rx.recv().await {
            let mut matched_cases = 0;

            for step in preview.steps {
                match matched_cases {
                    0 if step.op == Operation::Read => {
                        matched_cases += 1;
                    }
                    1 if step.op == Operation::CreateReplacement => {
                        matched_cases += 1;
                    }
                    2 if step.op == Operation::Replace => {
                        matched_cases += 1;
                    }
                    3 if step.op == Operation::CreateReplacement => {
                        matched_cases += 1;
                    }
                    4 if step.op == Operation::Replace => {
                        matched_cases += 1;
                    }
                    5 if step.op == Operation::DeleteReplaced => {
                        matched_cases += 1;
                    }
                    6 if step.op == Operation::DeleteReplaced => {
                        matched_cases += 1;
                    }
                    _ => {
                        println!("Skipped step in sequence {}: {:?}", matched_cases, step);
                    }
                }
            }

            assert!(
                matched_cases == 7,
                "Expected all steps to match, but found {} matched cases",
                matched_cases
            );
        }
    });

    let event_listener = tokio::spawn(async move {
        let mut summary_events = vec![];
        let mut cancel_events = vec![];
        let mut diagnostic_events = vec![];
        let mut resource_pre_events = vec![];
        let mut res_output_events = vec![];
        let mut prelude_events = vec![];

        let mut other_events = vec![];

        while let Some(event) = event_rx.recv().await {
            println!(
                "[TEST] Received event: {:?} at sequence: {:?}",
                event.event, event.sequence
            );
            match event.event {
                EventType::SummaryEvent { .. } => {
                    summary_events.push(event);
                }
                EventType::CancelEvent { .. } => {
                    cancel_events.push(event);
                }
                EventType::Diagnostic { .. } => {
                    diagnostic_events.push(event);
                }
                EventType::ResourcePreEvent { .. } => {
                    resource_pre_events.push(event);
                }
                EventType::ResOutputsEvent { .. } => {
                    res_output_events.push(event);
                }
                EventType::PreludeEvent { .. } => {
                    prelude_events.push(event);
                }
                _ => {
                    other_events.push(event);
                }
            }
        }

        assert!(!summary_events.is_empty(), "No summary events received");
        assert!(!cancel_events.is_empty(), "No cancel events received");
        assert!(
            !diagnostic_events.is_empty(),
            "No diagnostic events received"
        );
        assert!(
            !resource_pre_events.is_empty(),
            "No resource pre events received"
        );
        assert!(
            !res_output_events.is_empty(),
            "No resource output events received"
        );
        assert!(!prelude_events.is_empty(), "No prelude events received");

        assert!(
            other_events.is_empty(),
            "Unexpected events received: {:?}",
            other_events
        );
    });

    let operation = stack
        .up(
            StackUpOptions {
                show_replacement_steps: Some(true),
                show_reads: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx),
                event_tx,
            },
        )
        .await;

    assert!(operation.is_ok(), "Stack up failed: {:?}", operation);

    let (preview_result, event_result) = tokio::join!(preview_listener, event_listener);

    assert!(preview_result.is_ok(), "Preview listener failed");
    assert!(event_result.is_ok(), "Event listener failed");
}

#[tokio::test]
async fn test_up_fresh() {
    prep();
    let workspace = get_workspace_for_program("replacement");
    let stack = workspace
        .select_stack("fresh")
        .expect("Failed to select stack");

    let (preview_tx, mut preview_rx) = tokio::sync::mpsc::channel::<StackChangeSummary>(100);
    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);

    let preview_listener = tokio::spawn(async move {
        while let Some(preview) = preview_rx.recv().await {
            let mut matched_cases = 0;

            for step in preview.steps {
                match matched_cases {
                    0 if step.op == Operation::Create => {
                        matched_cases += 1;
                    }
                    1 if step.op == Operation::Create => {
                        matched_cases += 1;
                    }
                    _ => {
                        println!("Skipped step in sequence {}: {:?}", matched_cases, step);
                    }
                }
            }

            assert!(
                matched_cases == 2,
                "Expected all steps to match, but found {} matched cases",
                matched_cases
            );
        }
    });

    let event_listener = tokio::spawn(async move {
        let mut summary_events = vec![];
        let mut cancel_events = vec![];
        let mut diagnostic_events = vec![];
        let mut resource_pre_events = vec![];
        let mut res_output_events = vec![];
        let mut prelude_events = vec![];

        let mut other_events = vec![];

        while let Some(event) = event_rx.recv().await {
            println!(
                "[TEST] Received event: {:?} at sequence: {:?}",
                event.event, event.sequence
            );
            match event.event {
                EventType::SummaryEvent { .. } => {
                    summary_events.push(event);
                }
                EventType::CancelEvent { .. } => {
                    cancel_events.push(event);
                }
                EventType::Diagnostic { .. } => {
                    diagnostic_events.push(event);
                }
                EventType::ResourcePreEvent { .. } => {
                    resource_pre_events.push(event);
                }
                EventType::ResOutputsEvent { .. } => {
                    res_output_events.push(event);
                }
                EventType::PreludeEvent { .. } => {
                    prelude_events.push(event);
                }
                _ => {
                    other_events.push(event);
                }
            }
        }

        assert!(!summary_events.is_empty(), "No summary events received");
        assert!(!cancel_events.is_empty(), "No cancel events received");
        assert!(
            !diagnostic_events.is_empty(),
            "No diagnostic events received"
        );
        assert!(
            !resource_pre_events.is_empty(),
            "No resource pre events received"
        );
        assert!(
            !res_output_events.is_empty(),
            "No resource output events received"
        );
        assert!(!prelude_events.is_empty(), "No prelude events received");

        assert!(
            other_events.is_empty(),
            "Unexpected events received: {:?}",
            other_events
        );
    });

    let operation = stack
        .up(
            StackUpOptions {
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx),
                event_tx,
            },
        )
        .await;

    assert!(operation.is_ok(), "Stack up failed: {:?}", operation);

    let (preview_result, event_result) = tokio::join!(preview_listener, event_listener);

    assert!(preview_result.is_ok(), "Preview listener failed");
    assert!(event_result.is_ok(), "Event listener failed");
}

#[tokio::test]
async fn test_up_skip_preview() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let stack = workspace
        .select_stack("fresh")
        .expect("Failed to select stack");

    let (preview_tx, mut preview_rx) = tokio::sync::mpsc::channel::<StackChangeSummary>(100);
    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);

    let preview_listener = tokio::spawn(async move {
        while let Some(preview) = preview_rx.recv().await {
            assert!(
                false,
                "Expected no preview events, but received: {:?}",
                preview
            );
        }
    });

    let event_listener = tokio::spawn(async move {
        let mut summary_events = vec![];
        let mut cancel_events = vec![];
        let mut diagnostic_events = vec![];
        let mut resource_pre_events = vec![];
        let mut res_output_events = vec![];
        let mut prelude_events = vec![];

        let mut other_events = vec![];

        while let Some(event) = event_rx.recv().await {
            println!(
                "[TEST] Received event: {:?} at sequence: {:?}",
                event.event, event.sequence
            );
            match event.event {
                EventType::SummaryEvent { .. } => {
                    summary_events.push(event);
                }
                EventType::CancelEvent { .. } => {
                    cancel_events.push(event);
                }
                EventType::Diagnostic { .. } => {
                    diagnostic_events.push(event);
                }
                EventType::ResourcePreEvent { .. } => {
                    resource_pre_events.push(event);
                }
                EventType::ResOutputsEvent { .. } => {
                    res_output_events.push(event);
                }
                EventType::PreludeEvent { .. } => {
                    prelude_events.push(event);
                }
                _ => {
                    other_events.push(event);
                }
            }
        }

        assert!(!summary_events.is_empty(), "No summary events received");
        assert!(!cancel_events.is_empty(), "No cancel events received");
        assert!(
            !diagnostic_events.is_empty(),
            "No diagnostic events received"
        );
        assert!(
            !resource_pre_events.is_empty(),
            "No resource pre events received"
        );
        assert!(
            !res_output_events.is_empty(),
            "No resource output events received"
        );
        assert!(!prelude_events.is_empty(), "No prelude events received");

        assert!(
            other_events.is_empty(),
            "Unexpected events received: {:?}",
            other_events
        );
    });

    let operation = stack
        .up(
            StackUpOptions {
                skip_preview: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx),
                event_tx,
            },
        )
        .await;

    assert!(operation.is_ok(), "Stack up failed: {:?}", operation);

    let (preview_result, event_result) = tokio::join!(preview_listener, event_listener);

    assert!(preview_result.is_ok(), "Preview listener failed");
    assert!(event_result.is_ok(), "Event listener failed");
}

#[tokio::test]
async fn test_stack_destroy() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let stack = workspace
        .select_stack("case-1")
        .expect("Failed to select stack");

    let (preview_tx, mut preview_rx) = tokio::sync::mpsc::channel::<StackChangeSummary>(100);
    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);

    let preview_listener = tokio::spawn(async move {
        while let Some(preview) = preview_rx.recv().await {
            let mut matched_cases = 0;

            for step in preview.steps {
                match matched_cases {
                    0 if step.op == Operation::Delete => {
                        matched_cases += 1;
                    }
                    1 if step.op == Operation::Delete => {
                        matched_cases += 1;
                    }
                    _ => {
                        println!("Skipped step in sequence {}: {:?}", matched_cases, step);
                    }
                }
            }

            assert!(
                matched_cases == 2,
                "Expected all steps to match, but found {} matched cases",
                matched_cases
            );
        }
    });

    let event_listener = tokio::spawn(async move {
        let mut summary_events = vec![];
        let mut cancel_events = vec![];
        let mut resource_pre_events = vec![];
        let mut res_output_events = vec![];
        let mut prelude_events = vec![];

        let mut other_events = vec![];

        while let Some(event) = event_rx.recv().await {
            println!(
                "[TEST] Received event: {:?} at sequence: {:?}",
                event.event, event.sequence
            );
            match event.event {
                EventType::SummaryEvent { .. } => {
                    summary_events.push(event);
                }
                EventType::CancelEvent { .. } => {
                    cancel_events.push(event);
                }
                EventType::ResourcePreEvent { .. } => {
                    resource_pre_events.push(event);
                }
                EventType::ResOutputsEvent { .. } => {
                    res_output_events.push(event);
                }
                EventType::PreludeEvent { .. } => {
                    prelude_events.push(event);
                }
                _ => {
                    other_events.push(event);
                }
            }
        }

        assert!(!summary_events.is_empty(), "No summary events received");
        assert!(!cancel_events.is_empty(), "No cancel events received");
        assert!(
            !resource_pre_events.is_empty(),
            "No resource pre events received"
        );
        assert!(
            !res_output_events.is_empty(),
            "No resource output events received"
        );
        assert!(!prelude_events.is_empty(), "No prelude events received");

        assert!(
            other_events.is_empty(),
            "Unexpected events received: {:?}",
            other_events
        );
    });

    let operation = stack
        .destroy(
            StackDestroyOptions {
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx),
                event_tx,
            },
        )
        .await;

    assert!(operation.is_ok(), "Stack destroy failed: {:?}", operation);
    let (preview_result, event_result) = tokio::join!(preview_listener, event_listener);
    assert!(preview_result.is_ok(), "Preview listener failed");
    assert!(event_result.is_ok(), "Event listener failed");
}

#[tokio::test]
async fn test_stack_destroy_skip_preview() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let stack = workspace
        .select_stack("case-2")
        .expect("Failed to select stack");

    let (preview_tx, mut preview_rx) = tokio::sync::mpsc::channel::<StackChangeSummary>(100);
    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);

    let preview_listener = tokio::spawn(async move {
        while let Some(preview) = preview_rx.recv().await {
            assert!(
                false,
                "Expected no preview events, but received: {:?}",
                preview
            );
        }
    });

    let event_listener = tokio::spawn(async move {
        let mut summary_events = vec![];
        let mut cancel_events = vec![];
        let mut resource_pre_events = vec![];
        let mut res_output_events = vec![];
        let mut prelude_events = vec![];

        let mut other_events = vec![];

        while let Some(event) = event_rx.recv().await {
            println!(
                "[TEST] Received event: {:?} at sequence: {:?}",
                event.event, event.sequence
            );
            match event.event {
                EventType::SummaryEvent { .. } => {
                    summary_events.push(event);
                }
                EventType::CancelEvent { .. } => {
                    cancel_events.push(event);
                }
                EventType::ResourcePreEvent { .. } => {
                    resource_pre_events.push(event);
                }
                EventType::ResOutputsEvent { .. } => {
                    res_output_events.push(event);
                }
                EventType::PreludeEvent { .. } => {
                    prelude_events.push(event);
                }
                EventType::ProgressEvent { .. } => {
                    // ignore progress events for this test
                }
                _ => {
                    other_events.push(event);
                }
            }
        }

        assert!(!summary_events.is_empty(), "No summary events received");
        assert!(!cancel_events.is_empty(), "No cancel events received");
        assert!(
            !resource_pre_events.is_empty(),
            "No resource pre events received"
        );
        assert!(
            !res_output_events.is_empty(),
            "No resource output events received"
        );
        assert!(!prelude_events.is_empty(), "No prelude events received");

        assert!(
            other_events.is_empty(),
            "Unexpected events received: {:?}",
            other_events
        );
    });

    let operation = stack
        .destroy(
            StackDestroyOptions {
                skip_preview: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx),
                event_tx,
            },
        )
        .await;

    assert!(operation.is_ok(), "Stack destroy failed: {:?}", operation);
    let (preview_result, event_result) = tokio::join!(preview_listener, event_listener);
    assert!(preview_result.is_ok(), "Preview listener failed");
    assert!(event_result.is_ok(), "Event listener failed");
}

#[tokio::test]
async fn test_stack_refresh() {
    prep();
    let workspace = get_workspace_for_program("refresh");
    let stack = workspace
        .select_stack("case-1")
        .expect("Failed to select stack");

    let (preview_tx, mut preview_rx) = tokio::sync::mpsc::channel::<StackChangeSummary>(100);
    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);

    let preview_listener = tokio::spawn(async move {
        while let Some(preview) = preview_rx.recv().await {
            let mut matched_cases = 0;

            for step in preview.steps {
                match matched_cases {
                    0 if step.op == Operation::Refresh => {
                        matched_cases += 1;
                    }
                    1 if step.op == Operation::Refresh => {
                        matched_cases += 1;
                    }
                    _ => {
                        println!("Skipped step in sequence {}: {:?}", matched_cases, step);
                    }
                }
            }

            assert!(
                matched_cases == 2,
                "Expected all steps to match, but found {} matched cases",
                matched_cases
            );
        }
    });

    let event_listener = tokio::spawn(async move {
        let mut summary_events = vec![];
        let mut cancel_events = vec![];
        let mut resource_pre_events = vec![];
        let mut res_output_events = vec![];
        let mut prelude_events = vec![];

        let mut other_events = vec![];

        while let Some(event) = event_rx.recv().await {
            println!(
                "[TEST] Received event: {:?} at sequence: {:?}",
                event.event, event.sequence
            );
            match event.event {
                EventType::SummaryEvent { .. } => {
                    summary_events.push(event);
                }
                EventType::CancelEvent { .. } => {
                    cancel_events.push(event);
                }
                EventType::ResourcePreEvent { .. } => {
                    resource_pre_events.push(event);
                }
                EventType::ResOutputsEvent { .. } => {
                    res_output_events.push(event);
                }
                EventType::PreludeEvent { .. } => {
                    prelude_events.push(event);
                }
                _ => {
                    other_events.push(event);
                }
            }
        }

        assert!(!summary_events.is_empty(), "No summary events received");
        assert!(!cancel_events.is_empty(), "No cancel events received");
        assert!(
            !resource_pre_events.is_empty(),
            "No resource pre events received"
        );
        assert!(
            !res_output_events.is_empty(),
            "No resource output events received"
        );
        assert!(!prelude_events.is_empty(), "No prelude events received");
        assert!(
            other_events.is_empty(),
            "Unexpected events received: {:?}",
            other_events
        );
    });

    let operation = stack
        .refresh(
            StackRefreshOptions {
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx),
                event_tx,
            },
        )
        .await;

    assert!(operation.is_ok(), "Stack refresh failed: {:?}", operation);
    let (preview_result, event_result) = tokio::join!(preview_listener, event_listener);
    assert!(preview_result.is_ok(), "Preview listener failed");
    assert!(event_result.is_ok(), "Event listener failed");
}

#[tokio::test]
async fn test_stack_refresh_skip_preview() {
    prep();
    let workspace = get_workspace_for_program("refresh");
    let stack = workspace
        .select_stack("case-2")
        .expect("Failed to select stack");

    let (preview_tx, mut preview_rx) = tokio::sync::mpsc::channel::<StackChangeSummary>(100);
    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);

    let preview_listener = tokio::spawn(async move {
        while let Some(preview) = preview_rx.recv().await {
            assert!(
                false,
                "Expected no preview events, but received: {:?}",
                preview
            );
        }
    });

    let event_listener = tokio::spawn(async move {
        let mut summary_events = vec![];
        let mut cancel_events = vec![];
        let mut resource_pre_events = vec![];
        let mut res_output_events = vec![];
        let mut prelude_events = vec![];

        let mut other_events = vec![];

        while let Some(event) = event_rx.recv().await {
            println!(
                "[TEST] Received event: {:?} at sequence: {:?}",
                event.event, event.sequence
            );
            match event.event {
                EventType::SummaryEvent { .. } => {
                    summary_events.push(event);
                }
                EventType::CancelEvent { .. } => {
                    cancel_events.push(event);
                }
                EventType::ResourcePreEvent { .. } => {
                    resource_pre_events.push(event);
                }
                EventType::ResOutputsEvent { .. } => {
                    res_output_events.push(event);
                }
                EventType::PreludeEvent { .. } => {
                    prelude_events.push(event);
                }
                EventType::ProgressEvent { .. } => {
                    // ignore progress events for this test
                }
                _ => {
                    other_events.push(event);
                }
            }
        }

        assert!(!summary_events.is_empty(), "No summary events received");
        assert!(!cancel_events.is_empty(), "No cancel events received");
        assert!(
            !resource_pre_events.is_empty(),
            "No resource pre events received"
        );
        assert!(
            !res_output_events.is_empty(),
            "No resource output events received"
        );
        assert!(!prelude_events.is_empty(), "No prelude events received");

        assert!(
            other_events.is_empty(),
            "Unexpected events received: {:?}",
            other_events
        );
    });

    let operation = stack
        .refresh(
            StackRefreshOptions {
                skip_preview: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx),
                event_tx,
            },
        )
        .await;

    assert!(operation.is_ok(), "Stack refresh failed: {:?}", operation);
    let (preview_result, event_result) = tokio::join!(preview_listener, event_listener);
    assert!(preview_result.is_ok(), "Preview listener failed");
    assert!(event_result.is_ok(), "Event listener failed");
}
