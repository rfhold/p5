use pulumi_automation::{
    local::LocalWorkspace,
    workspace::{ConfigValue, StackListOptions, StackRemoveOptions, Workspace},
};

const FIXTURES_DIR: &str = "tests/fixtures";
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

#[test]
fn fixtures_exist() {
    assert!(
        std::path::Path::new(FIXTURES_DIR).exists(),
        "The fixtures directory does not exist at the expected path: {}",
        FIXTURES_DIR
    );
}

#[test]
fn set_env_vars() {
    prep();
    assert!(
        std::env::var("PULUMI_BACKEND_URL").is_ok(),
        "PULUMI_BACKEND_URL is not set"
    );
    assert!(
        std::env::var("PULUMI_CONFIG_PASSPHRASE_FILE").is_ok(),
        "PULUMI_CONFIG_PASSPHRASE_FILE is not set"
    );
}

#[test]
fn new_stack() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    workspace
        .create_stack("new_stack", Default::default())
        .expect("Failed to create stack");
}

#[test]
fn select_stack() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    workspace
        .select_stack("existing")
        .expect("Failed to create stack");
}

#[test]
fn select_non_existent_stack() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let result = workspace.select_stack("non-existent");
    assert!(
        result.is_err(),
        "Expected error when selecting a non-existent stack"
    );
}

#[test]
fn whoami() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let whoami = workspace.whoami().expect("Failed to get whoami");

    let backend_url = std::env::var("PULUMI_BACKEND_URL").expect("PULUMI_BACKEND_URL is not set");

    assert!(whoami.url.is_some(), "Expected whoami URL to be set");

    assert_eq!(
        whoami.url.unwrap(),
        backend_url,
        "Expected backend URL to match the environment variable"
    );
}

#[test]
fn get_config() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let config_value = workspace
        .get_config("configured", "json", false)
        .expect("Failed to get config");

    assert_eq!(
        config_value.value, "{\"array\":[\"one\",\"two\"],\"key\":\"value\"}",
        "Expected config value to match"
    );
}

#[test]
fn get_config_path() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let config_value = workspace
        .get_config("configured", "basic-program:json", true)
        .expect("Failed to get config");

    assert_eq!(
        config_value.value, "{\"array\":[\"one\",\"two\"],\"key\":\"value\"}",
        "Expected config value to match"
    );
}

#[test]
fn get_config_secret() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let config_value = workspace
        .get_config("configured", "secret", false)
        .expect("Failed to get config");

    assert_eq!(
        config_value.value, "SUPER_SECRET",
        "Expected secret config value to match"
    );
}

#[test]
fn get_and_set_config() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let config_value = workspace
        .get_config("configured", "foo", false)
        .expect("Failed to get config");

    assert_eq!(config_value.value, "bar", "Expected config value to match");

    workspace
        .set_config(
            "configured",
            "foo",
            false,
            ConfigValue {
                value: "biz".to_string(),
                secret: false,
            },
        )
        .expect("Failed to set config");

    let updated_config_value = workspace
        .get_config("configured", "foo", false)
        .expect("Failed to get updated config");

    assert_eq!(
        updated_config_value.value, "biz",
        "Expected updated config value to match"
    );
}

#[test]
fn get_and_set_config_bool() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let config_value = workspace
        .get_config("configured", "bool", false)
        .expect("Failed to get config");

    assert_eq!(config_value.value, "true", "Expected config value to match");

    workspace
        .set_config(
            "configured",
            "bool",
            false,
            ConfigValue {
                value: "false".to_string(),
                secret: false,
            },
        )
        .expect("Failed to set config");

    let updated_config_value = workspace
        .get_config("configured", "bool", false)
        .expect("Failed to get updated config");

    assert_eq!(
        updated_config_value.value, "false",
        "Expected updated config value to match"
    );
}

#[test]
fn remove_config_key() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let config_value = workspace
        .get_config("configured", "to-remove", false)
        .expect("Failed to get config");

    assert_eq!(config_value.value, "true", "Expected config value to match");

    workspace
        .remove_config("configured", "to-remove", false)
        .expect("Failed to set config");

    let new_result = workspace.get_config("configured", "to-remove", false);

    assert!(new_result.is_err(), "Expected to-remove to no longer exist");
}

#[test]
fn stack_outputs() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let outputs = workspace
        .stack_outputs("configured")
        .expect("Failed to get stack outputs");

    assert!(
        outputs.contains_key("secret"),
        "Expected stack outputs to contain 'secret'"
    );

    assert!(
        outputs.contains_key("json"),
        "Expected stack outputs to contain 'json'"
    );

    assert_eq!(
        outputs.get("secret").unwrap(),
        "SUPER_SECRET",
        "Expected secret output value to match"
    );
}

#[test]
fn list_program_stacks() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let stacks = workspace.list_stacks(None).expect("Failed to list stacks");

    assert!(
        !stacks.is_empty(),
        "Expected at least one stack to be present"
    );

    let expected_stacks = vec!["existing", "configured", "fresh"];

    for expected_stack in expected_stacks {
        assert!(
            stacks.iter().any(|s| s.name == expected_stack),
            "Expected stack '{}' to be present in the list",
            expected_stack
        );
    }
}

#[test]
fn list_all_stacks() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let stacks = workspace
        .list_stacks(Some(StackListOptions { all: Some(true) }))
        .expect("Failed to list all stacks");

    assert!(
        !stacks.is_empty(),
        "Expected at least one stack to be present"
    );

    let expected_stacks = vec!["existing", "configured", "organization/replacement/test"];

    for expected_stack in expected_stacks {
        assert!(
            stacks.iter().any(|s| s.name == expected_stack),
            "Expected stack '{}' to be present in the list",
            expected_stack
        );
    }
}

#[test]
fn remove_empty_stack() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    workspace
        .remove_stack("empty", None)
        .expect("Failed to remove empty stack");

    let stacks = workspace
        .list_stacks(None)
        .expect("Failed to list stacks after removing empty stack");

    assert!(
        !stacks.iter().any(|s| s.name == "empty"),
        "Expected 'empty' stack to be removed"
    );
}

#[test]
fn remove_stack_forced() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    assert!(
        workspace.remove_stack("to-remove-forced", None).is_err(),
        "Expected error when trying to remove a stack that is not empty without force"
    );

    let stacks = workspace
        .list_stacks(None)
        .expect("Failed to list stacks after removing empty stack");

    assert!(
        stacks.iter().any(|s| s.name == "to-remove-forced"),
        "Expected 'to-remove-forced' stack to still exist before forced removal"
    );

    assert!(
        workspace
            .remove_stack(
                "to-remove-forced",
                Some(StackRemoveOptions {
                    force: Some(true),
                    preserve_config: Some(false)
                })
            )
            .is_ok(),
        "Failed to remove stack with force"
    );

    let stacks = workspace
        .list_stacks(None)
        .expect("Failed to list stacks after forced removal");

    assert!(
        !stacks.iter().any(|s| s.name == "to-remove-forced"),
        "Expected 'to-remove-forced' stack to be removed after forced removal"
    );
}

#[test]
fn export_stack_import_stack() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let deployment = workspace
        .export_stack("existing")
        .expect("Failed to export stack");

    assert_eq!(deployment.version, 3, "Expected deployment version to be 1");
    assert!(
        deployment.deployment.is_object(),
        "Expected deployment to be an object"
    );

    workspace
        .import_stack("existing", deployment)
        .expect("Failed to import stack");
}

#[test]
fn import_stack_forced() {
    prep();
    let workspace = get_workspace_for_program("basic-program");
    let deployment = workspace
        .export_stack("existing")
        .expect("Failed to export stack");

    workspace
        .create_stack("existing-copy", Default::default())
        .expect("Failed to create existing-copy stack");

    assert!(
        workspace.import_stack("existing-copy", deployment).is_err(),
        "Expected error when importing a stack that already exists without force"
    );
}
