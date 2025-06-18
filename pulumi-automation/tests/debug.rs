use std::collections::HashMap;

use pulumi_automation::{
    event::{EngineEvent, EventType},
    local::LocalWorkspace,
    stack::{
        Operation, PulumiProcessListener, Stack, StackChangeSummary, StackDestroyOptions,
        StackRefreshOptions, StackUpOptions,
    },
    workspace::{StackCreateOptions, Workspace},
};
use util::shake_json_paths;

mod util;

const PROGRAMS_PATH: &str = "tests/fixtures/programs";
const DUMP_PATH: &str = "tests/fixtures/dumps";
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

fn dump_pulumi_json(program: &str, stack_name: &str) -> Result<(), Box<dyn std::error::Error>> {
    let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR not set");
    let program_path = std::path::Path::new(&manifest_dir)
        .join(PROGRAMS_PATH)
        .join(program);

    let dump_path = std::path::Path::new(&manifest_dir).join(DUMP_PATH);
    if !dump_path.exists() {
        std::fs::create_dir_all(&dump_path)?;
    }

    let std_out_file = dump_path.join(format!("{}_{}.stdout.json", stack_name, program));
    let std_err_file = dump_path.join(format!("{}_{}.stderr.json", stack_name, program));

    let cwd = program_path.to_str().unwrap();

    std::process::Command::new("pulumi")
        .arg("up")
        .arg("--stack")
        .arg(stack_name)
        .arg("--cwd")
        .arg(cwd)
        .arg("--json")
        .arg("--yes")
        .stdout(std::fs::File::create(&std_out_file)?)
        .stderr(std::fs::File::create(&std_err_file)?)
        .spawn()?;

    Ok(())
}

#[tokio::test]
async fn test_event_type_extra_values() {
    prep();
    dump_pulumi_json("debug", "base-reference")
        .expect("Failed to dump Pulumi JSON for debug stack");
    let workspace = get_workspace_for_program("debug");
    let stack = workspace
        .select_or_create_stack(
            "base",
            Some(StackCreateOptions {
                copy_config_from: Some("base".to_string()),
                ..Default::default()
            }),
        )
        .expect("Failed to select stack");

    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);
    let event_listener = tokio::spawn(async move {
        let mut non_empty_values: HashMap<String, Vec<serde_json::Value>> = HashMap::new();
        let mut total_events = 0;
        while let Some(event) = event_rx.recv().await {
            total_events += 1;
            let event_type = event.event.to_string();
            if !non_empty_values.contains_key(&event_type) {
                non_empty_values.insert(event_type.clone(), vec![]);
            }

            let extra_values = match &event.event {
                EventType::CancelEvent { details } => details.extra_values.clone(),
                EventType::Diagnostic { details } => details.extra_values.clone(),
                EventType::PreludeEvent { details } => details.extra_values.clone(),
                EventType::SummaryEvent { details } => details.extra_values.clone(),
                EventType::ResourcePreEvent { details } => details.extra_values.clone(),
                EventType::ResOutputsEvent { details } => details.extra_values.clone(),
                EventType::StdoutEvent { details } => details.extra_values.clone(),
                EventType::ResOpFailedEvent { details } => details.extra_values.clone(),
                EventType::PolicyEvent { details } => details.extra_values.clone(),
                EventType::StartDebuggingEvent { details } => details.extra_values.clone(),
                EventType::ProgressEvent { details } => details.extra_values.clone(),
                EventType::Unknown { extra } => {
                    assert!(false, "Unknown event type received: {:?}", extra);
                    None
                }
            };

            if let Some(values) = extra_values {
                non_empty_values.get_mut(&event_type).unwrap().push(values);
            }
        }

        assert!(total_events > 0, "No events received during test");

        let mut has_non_empty_values = false;
        let mut unique_event_types = 0;
        for (key, value) in &non_empty_values {
            unique_event_types += 1;
            let paths = shake_json_paths(value.to_vec());
            println!("Event type: {}", key);
            for path in paths {
                has_non_empty_values = true;
                println!("  Path: {}", path);
            }
        }

        assert!(
            !has_non_empty_values,
            "Non-empty extra values found in events"
        );

        assert!(
            unique_event_types == 11,
            "Expected 11 event types, found: {}",
            unique_event_types
        );
    });

    stack
        .up(
            StackUpOptions {
                skip_preview: Some(true),
                show_replacement_steps: Some(true),
                show_reads: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: None,
                event_tx: event_tx.clone(),
            },
        )
        .await
        .expect("Stack up failed");

    stack
        .refresh(
            StackRefreshOptions {
                skip_preview: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: None,
                event_tx: event_tx.clone(),
            },
        )
        .await
        .expect("Stack refresh failed");

    stack
        .destroy(
            StackDestroyOptions {
                skip_preview: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: None,
                event_tx: event_tx,
            },
        )
        .await
        .expect("Stack destroy failed");

    let result = event_listener.await;

    assert!(result.is_ok(), "Event listener failed: {:?}", result);
}

#[tokio::test]
async fn test_event_type_extra_values_failure() {
    prep();
    dump_pulumi_json("debug", "fail-reference")
        .expect("Failed to dump Pulumi JSON for failure reference");
    let workspace = get_workspace_for_program("debug");
    let stack = workspace
        .select_or_create_stack(
            "fail",
            Some(StackCreateOptions {
                copy_config_from: Some("fail".to_string()),
                ..Default::default()
            }),
        )
        .expect("Failed to select stack");

    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);
    let event_listener = tokio::spawn(async move {
        let mut non_empty_values: HashMap<String, Vec<serde_json::Value>> = HashMap::new();
        let mut total_events = 0;
        while let Some(event) = event_rx.recv().await {
            total_events += 1;
            let event_type = event.event.to_string();
            if !non_empty_values.contains_key(&event_type) {
                non_empty_values.insert(event_type.clone(), vec![]);
            }

            let extra_values = match &event.event {
                EventType::CancelEvent { details } => details.extra_values.clone(),
                EventType::Diagnostic { details } => details.extra_values.clone(),
                EventType::PreludeEvent { details } => details.extra_values.clone(),
                EventType::SummaryEvent { details } => details.extra_values.clone(),
                EventType::ResourcePreEvent { details } => details.extra_values.clone(),
                EventType::ResOutputsEvent { details } => details.extra_values.clone(),
                EventType::StdoutEvent { details } => details.extra_values.clone(),
                EventType::ResOpFailedEvent { details } => details.extra_values.clone(),
                EventType::PolicyEvent { details } => details.extra_values.clone(),
                EventType::StartDebuggingEvent { details } => details.extra_values.clone(),
                EventType::ProgressEvent { details } => details.extra_values.clone(),
                EventType::Unknown { extra } => {
                    assert!(false, "Unknown event type received: {:?}", extra);
                    None
                }
            };

            if let Some(values) = extra_values {
                non_empty_values.get_mut(&event_type).unwrap().push(values);
            }
        }

        assert!(total_events > 0, "No events received during test");

        let mut has_non_empty_values = false;
        let mut unique_event_types = 0;
        for (key, value) in &non_empty_values {
            unique_event_types += 1;
            let paths = shake_json_paths(value.to_vec());
            println!("Event type: {}", key);
            for path in paths {
                has_non_empty_values = true;
                println!("  Path: {}", path);
            }
        }

        assert!(
            !has_non_empty_values,
            "Non-empty extra values found in events"
        );

        assert!(
            unique_event_types == 11,
            "Expected 11 event types, found: {}",
            unique_event_types
        );
    });

    stack
        .up(
            StackUpOptions {
                skip_preview: Some(true),
                show_replacement_steps: Some(true),
                show_reads: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: None,
                event_tx: event_tx.clone(),
            },
        )
        .await
        .expect("Stack up failed");

    stack
        .refresh(
            StackRefreshOptions {
                skip_preview: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: None,
                event_tx: event_tx.clone(),
            },
        )
        .await
        .expect("Stack refresh failed");

    stack
        .destroy(
            StackDestroyOptions {
                skip_preview: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: None,
                event_tx: event_tx,
            },
        )
        .await
        .expect("Stack destroy failed");

    let result = event_listener.await;

    assert!(result.is_ok(), "Event listener failed: {:?}", result);
}

#[tokio::test]
async fn test_preview_extra_values() {
    prep();
    let workspace = get_workspace_for_program("debug");
    let stack = workspace
        .select_or_create_stack(
            "preview",
            Some(StackCreateOptions {
                copy_config_from: Some("preview".to_string()),
                ..Default::default()
            }),
        )
        .expect("Failed to select stack");

    let (preview_tx, mut preview_rx) = tokio::sync::mpsc::channel::<StackChangeSummary>(100);
    let (event_tx, mut event_rx) = tokio::sync::mpsc::channel::<EngineEvent>(100);

    let preview_listener = tokio::spawn(async move {
        let mut non_empty_values: HashMap<Operation, Vec<serde_json::Value>> = HashMap::new();
        let mut total_previews = 0;
        let mut total_preview_steps = 0;

        while let Some(preview) = preview_rx.recv().await {
            total_previews += 1;
            for step in &preview.steps {
                total_preview_steps += 1;
                let step_type = step.op.clone();
                if !non_empty_values.contains_key(&step_type) {
                    non_empty_values.insert(step_type.clone(), vec![]);
                }

                if let Some(extra_values) = &step.extra_values {
                    non_empty_values
                        .get_mut(&step_type)
                        .unwrap()
                        .push(extra_values.clone());
                }
            }
        }

        assert!(total_previews > 0, "No previews received during test");

        assert!(
            total_preview_steps > 0,
            "No preview steps received during test"
        );

        let mut has_non_empty_values = false;
        let mut unique_operations = 0;
        let mut all_values = vec![];
        for (key, value) in &non_empty_values {
            unique_operations += 1;
            all_values.extend(value.to_vec());
            let paths = shake_json_paths(value.to_vec());
            println!("Operation: {:?}", key);
            for path in paths {
                has_non_empty_values = true;
                println!("  Path: {}", path);
            }
        }

        println!("All Operation Paths:");
        let paths = shake_json_paths(all_values.clone());
        for path in paths {
            println!("  Path: {}", path);
        }

        assert!(
            !has_non_empty_values,
            "Non-empty extra values found in previews"
        );
        assert!(
            unique_operations == 15,
            "Expected 15 unique operations, found: {}",
            unique_operations
        );
    });

    let event_listener = tokio::spawn(async move {
        let mut total_events = 0;
        while let Some(_event) = event_rx.recv().await {
            total_events += 1;
        }

        assert!(total_events > 0, "No events received during test");
    });

    stack
        .up(
            StackUpOptions {
                show_replacement_steps: Some(true),
                show_reads: Some(true),
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx.clone()),
                event_tx: event_tx.clone(),
            },
        )
        .await
        .expect("Stack up failed");

    stack
        .refresh(
            StackRefreshOptions {
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx.clone()),
                event_tx: event_tx.clone(),
            },
        )
        .await
        .expect("Stack refresh failed");

    stack
        .destroy(
            StackDestroyOptions {
                ..Default::default()
            },
            PulumiProcessListener {
                preview_tx: Some(preview_tx),
                event_tx: event_tx,
            },
        )
        .await
        .expect("Stack destroy failed");

    let (result_preview_listener, result_event_listener) =
        tokio::join!(preview_listener, event_listener);

    assert!(result_preview_listener.is_ok(), "Preview listener failed");
    assert!(result_event_listener.is_ok(), "Event listener failed");
}
