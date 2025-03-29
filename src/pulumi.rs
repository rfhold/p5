use std::{collections::HashMap, process::Command};

use serde::Deserialize;
use walkdir::WalkDir;

#[derive(Debug, Deserialize)]
pub struct Credentials {
    pub current: String,
}

pub fn get_credentials() -> Result<Credentials, Box<dyn std::error::Error>> {
    let home = std::env::var("HOME")?;
    let credentials_path = format!("{}/.pulumi/credentials.json", home);
    let credentials_content = std::fs::read_to_string(credentials_path)?;
    let credentials: Credentials = serde_json::from_str(&credentials_content)?;

    Ok(credentials)
}

// look for all programs by ./**{max_depth}/Pulumi.yaml
pub fn find_programs(max_depth: i32) -> Result<Vec<Program>, Box<dyn std::error::Error>> {
    let mut programs = Vec::new();

    // Convert max_depth to usize (if negative, default to 0)
    let max_depth = if max_depth < 0 { 0 } else { max_depth as usize };

    let cwd = std::env::current_dir()?;

    // WalkDir::new(".") to iterate from the current directory
    // Set the max_depth search accordingly.
    for entry in WalkDir::new(".")
        .max_depth(max_depth)
        .into_iter()
        .filter_map(|e| e.ok())
    {
        // Check if it's a file and if its filename is "Pulumi.yaml"
        if entry.file_type().is_file() && entry.file_name() == "Pulumi.yaml" {
            let path = entry.path().to_path_buf();
            let config = read_program_config(&path)?;
            let rel_path = path
                .display()
                .to_string()
                .trim_start_matches(".")
                .trim_end_matches("Pulumi.yaml")
                .to_string();
            let abs_path = format!("{}{}", cwd.display(), rel_path);
            programs.push(Program {
                path: abs_path,
                config,
            });
        }
    }

    Ok(programs)
}

// look for all {cwd}/Pulumi.*.yaml and return the vector of *
pub fn find_stack_files(cwd: &str) -> Result<Vec<String>, Box<dyn std::error::Error>> {
    let mut stack_files = Vec::new();

    for entry in WalkDir::new(cwd)
        .into_iter()
        .filter_map(|e| e.ok())
        .filter(|e| e.file_type().is_file())
        .filter(|e| e.file_name() != "Pulumi.yaml")
    {
        if let Some(extension) = entry.path().extension() {
            if extension == "yaml" {
                let file_name = entry.file_name().to_str().unwrap().to_string();
                stack_files.push(
                    file_name
                        .trim_start_matches("Pulumi.")
                        .trim_end_matches(".yaml")
                        .to_string(),
                );
            }
        }
    }

    Ok(stack_files)
}

fn read_program_config(
    path: &std::path::PathBuf,
) -> Result<ProgramConfig, Box<dyn std::error::Error>> {
    let content = std::fs::read_to_string(path)?;
    let config: ProgramConfig = serde_yaml::from_str(&content)?;

    Ok(config)
}

#[derive(Debug)]
pub struct Program {
    pub path: String,
    pub config: ProgramConfig,
}

#[derive(Debug, Deserialize)]
pub struct ProgramConfig {
    pub name: String,
}

#[derive(Debug, Deserialize)]
pub struct StackManifest {
    pub time: String,
    pub magic: String,
    pub version: String,
}

#[derive(Debug, Deserialize)]
pub struct StackResource {
    pub urn: String,
    pub name: Option<String>,
    pub custom: bool,
    pub type_: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct StackDeployment {
    pub manifest: StackManifest,
    pub resources: Option<Vec<StackResource>>,
}

#[derive(Debug, Deserialize)]
pub struct StackExport {
    pub version: i32,
    pub deployment: StackDeployment,
}

#[derive(Debug, Deserialize)]
pub struct Stack {
    pub name: String,
    pub current: bool,
    pub last_update: Option<String>,
    pub resource_count: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct StackPreview {
    pub config: HashMap<String, String>,
    pub steps: Vec<StackStep>,
    pub duration: i64,
    pub change_summary: Option<StackChangeSummary>,
}

#[derive(Debug, Deserialize)]
pub struct StackStep {
    pub op: String,
    pub urn: String,
    pub old_state: Option<StackResource>,
    pub new_state: Option<StackResource>,
}

#[derive(Debug, Deserialize)]
pub struct StackChangeSummary {
    pub create: i32,
    pub update: i32,
    pub discard: i32,
    pub delete: i32,
    pub same: i32,
}

#[derive(Debug, Deserialize, Clone)]
pub struct Provider {
    pub name: String,
    pub urn: String,
}

pub trait ProgramBackend {
    fn init_stack(
        &self,
        stack_name: &str,
        secrets_provider: &str,
    ) -> Result<(), Box<dyn std::error::Error>>;
    fn list_stacks(&self) -> Result<Vec<Stack>, Box<dyn std::error::Error>>;
    fn list_stack_resources(
        &self,
        stack_name: &str,
    ) -> Result<Vec<StackResource>, Box<dyn std::error::Error>>;
    fn import_stack_resource(
        &self,
        stack_name: &str,
        resource_type: &str,
        resource_name: &str,
        resource_id: &str,
        provider: Option<Provider>,
    ) -> Result<(), Box<dyn std::error::Error>>;
    fn set_cwd(&mut self, cwd: String);
    fn preview_stack(&self, stack_name: &str) -> Result<StackPreview, Box<dyn std::error::Error>>;
    fn refresh_stack(&self, stack_name: &str) -> Result<(), Box<dyn std::error::Error>>;
    fn delete_stack(&self, stack_name: &str) -> Result<(), Box<dyn std::error::Error>>;
    fn update_stack(&self, stack_name: &str) -> Result<(), Box<dyn std::error::Error>>;
    fn delete_stack_resource(
        &self,
        stack: &str,
        resource_urn: &str,
    ) -> Result<(), Box<dyn std::error::Error>>;
}

pub struct PulumiContext {
    pub cwd: String,
    pub backend_url: Option<String>,
    pub env: Vec<(String, String)>,
}

pub struct LocalProgramBackend {
    pub context: PulumiContext,
}

impl LocalProgramBackend {
    pub fn new(context: PulumiContext) -> LocalProgramBackend {
        LocalProgramBackend { context }
    }

    fn configure_environment(&self) -> Result<(), Box<dyn std::error::Error>> {
        let backend = self.context.backend_url.clone();

        if let Some(backend) = backend {
            let credentials = get_credentials();

            if credentials.is_err() || credentials.unwrap().current != backend {
                Command::new("pulumi").arg("login").arg(backend).status()?;
            }
        }

        Ok(())
    }

    fn run_pulumi_command(&self, args: Vec<&str>) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
        self.configure_environment()?;

        let output = Command::new("pulumi")
            .arg("--non-interactive")
            .arg("--color=never")
            .arg("--cwd")
            .arg(&self.context.cwd)
            .args(args)
            .current_dir(&self.context.cwd)
            .output()?;

        if !output.status.success() {
            return Err(format!(
                "Failed to run pulumi: {}",
                String::from_utf8_lossy(&output.stderr)
            )
            .into());
        }

        Ok(output.stdout)
    }

    pub fn cwd(&self) -> &str {
        &self.context.cwd
    }
}

impl ProgramBackend for LocalProgramBackend {
    fn init_stack(
        &self,
        stack_name: &str,
        secrets_provider: &str,
    ) -> Result<(), Box<dyn std::error::Error>> {
        self.run_pulumi_command(vec![
            "stack",
            "init",
            stack_name,
            "--secrets-provider",
            secrets_provider,
        ])?;

        Ok(())
    }

    fn list_stacks(&self) -> Result<Vec<Stack>, Box<dyn std::error::Error>> {
        let output = self.run_pulumi_command(vec!["stack", "ls", "--json"])?;

        let stacks: Vec<Stack> = serde_json::from_slice(&output)?;

        Ok(stacks)
    }

    fn list_stack_resources(
        &self,
        stack_name: &str,
    ) -> Result<Vec<StackResource>, Box<dyn std::error::Error>> {
        let output = self.run_pulumi_command(vec!["--stack", stack_name, "stack", "export"])?;

        let stack_export: StackExport = serde_json::from_slice(&output)?;

        Ok(stack_export
            .deployment
            .resources
            .or(Some(Vec::new()))
            .unwrap())
    }

    fn import_stack_resource(
        &self,
        stack_name: &str,
        resource_type: &str,
        resource_name: &str,
        resource_id: &str,
        provider: Option<Provider>,
    ) -> Result<(), Box<dyn std::error::Error>> {
        let mut args = vec![
            "--stack",
            stack_name,
            "import",
            resource_type,
            resource_name,
            resource_id,
            "--yes",
        ];

        let provider_arg = match provider {
            Some(provider) => format!("{}={}", provider.name, provider.urn),
            None => "".to_string(),
        };

        if !provider_arg.is_empty() {
            args.push("--provider");
            args.push(&provider_arg);
        }

        self.run_pulumi_command(args)?;

        Ok(())
    }

    fn set_cwd(&mut self, cwd: String) {
        self.context.cwd = cwd;
    }

    fn refresh_stack(&self, stack_name: &str) -> Result<(), Box<dyn std::error::Error>> {
        self.run_pulumi_command(vec!["--stack", stack_name, "refresh", "--yes"])?;

        Ok(())
    }

    fn preview_stack(&self, stack_name: &str) -> Result<StackPreview, Box<dyn std::error::Error>> {
        let output = self.run_pulumi_command(vec!["--stack", stack_name, "preview", "--json"])?;

        let stack_preview: StackPreview = serde_json::from_slice(&output)?;

        Ok(stack_preview)
    }

    fn update_stack(&self, stack_name: &str) -> Result<(), Box<dyn std::error::Error>> {
        self.run_pulumi_command(vec![
            "--stack",
            stack_name,
            "up",
            "--skip-preview",
            "--yes",
            "--json",
        ])?;

        Ok(())
    }

    fn delete_stack(&self, stack_name: &str) -> Result<(), Box<dyn std::error::Error>> {
        self.run_pulumi_command(vec![
            "stack",
            "rm",
            stack_name,
            "--yes",
            "--force",
            "--preserve-config",
        ])?;

        Ok(())
    }

    fn delete_stack_resource(
        &self,
        stack: &str,
        resource_urn: &str,
    ) -> Result<(), Box<dyn std::error::Error>> {
        self.run_pulumi_command(vec![
            "--stack",
            stack,
            "state",
            "delete",
            resource_urn,
            "--force",
            "--yes",
        ])?;

        Ok(())
    }
}
