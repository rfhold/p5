use std::{collections::HashMap, path::Path};

use serde::{Deserialize, Serialize};
use strum::Display;

pub trait Interactor {
    type Error;

    fn stack_list(&self, options: Vec<StackListOption>) -> Result<Vec<Stack>, Self::Error>;
    fn stack_export(
        &self,
        stack_name: &str,
        options: Vec<StackExportOption>,
    ) -> Result<StackExport, Self::Error>;
    fn stack_output(
        &self,
        stack_name: &str,
        options: Vec<StackOutputOption>,
    ) -> Result<serde_json::Value, Self::Error>;
    fn stack_preview(
        &self,
        stack_name: &str,
        options: Vec<StackPreviewOption>,
    ) -> Result<StackPreview, Self::Error>;
    // fn stack_init(
    //     &self,
    //     stack_name: &str,
    //     options: Vec<StackInitOption>,
    // ) -> Result<Self::Result, Self::Error>;
    // fn stack_delete(
    //     &self,
    //     stack_name: &str,
    //     options: Vec<StackDeleteOption>,
    // ) -> Result<Self::Result, Self::Error>;
    // fn destroy(
    //     &self,
    //     stack_name: &str,
    //     options: Vec<DestroyOption>,
    // ) -> Result<Self::Result, Self::Error>;
    // fn up(&self, stack_name: &str, options: Vec<UpOption>) -> Result<Self::Result, Self::Error>;
    // fn refresh(
    //     &self,
    //     stack_name: &str,
    //     options: Vec<RefreshOption>,
    // ) -> Result<Self::Result, Self::Error>;
    // fn state_delete(
    //     &self,
    //     stack_name: &str,
    //     resource_urn: &str,
    //     options: Vec<StateDeleteOption>,
    // ) -> Result<Self::Result, Self::Error>;
    // fn state_move(
    //     &self,
    //     source_stack_name: &str,
    //     target_stack_name: &str,
    //     resource_urn: &str,
    //     options: Vec<StateMoveOption>,
    // ) -> Result<Self::Result, Self::Error>;
    // fn state_repair(&self, stack_name: &str) -> Result<Self::Result, Self::Error>;
    // fn state_rename(
    //     &self,
    //     stack_name: &str,
    //     resource_urn: &str,
    //     new_name: &str,
    // ) -> Result<Self::Result, Self::Error>;
    // fn import(
    //     &self,
    //     stack_name: &str,
    //     resource_type: &str,
    //     resource_name: &str,
    //     id: &str,
    //     options: Vec<ImportOption>,
    // ) -> Result<Self::Result, Self::Error>;
}

#[derive(Debug, Clone, Deserialize, Default)]
pub struct Stack {
    pub name: String,
    pub current: bool,
    pub last_update: Option<String>,
    pub resource_count: Option<u32>,
}

pub enum StackInitOption {
    SecretsProvider(String),
    CopyFrom(String),
}

pub enum StackDeleteOption {
    Force,
    PreserveConfig,
}

pub enum StackSelectOption {
    SecretsProvider(String),
    Create,
}

pub enum StackOutputOption {
    ShowSecrets(bool),
}

pub enum DestroyOption {
    ContinueOnError,
    Diff,
    ExcludeProtected,
    Message(String),
    Parallel(u32),
    PreviewOnly,
    Refresh(bool),
    Remove,
    ShowConfig, // TODO: Check
    ShowReplacementSteps,
    ShowSames,
    Target(String),
    TargetDependents,
}

pub enum UpOption {
    AttachDebugger,
    Config(String),
    ContinueOnError,
    Diff,
    ExpectNoChanges,
    Message(String),
    Parallel(u32),
    Refresh(bool),
    Replace(String),
    SecretsProvider(String),
    ShowConfig, // TODO: Check
    ShowFullOutput,
    ShowReads,
    ShowSames,
    ShowSecrets,
    Target(String),
    TargetDependents,
    TargetReplace(String),
}

pub enum RefreshOption {
    ClearPendingCreates,
    Diff,
    ExpectNoChanges,
    ImportPendingCreates(String),
    Message(String),
    Parallel(u32),
    PreviewOnly,
    ShowReplacementSteps,
    ShowSames,
    SkipPendingCreates,
    Target(String),
}

#[derive(Debug, Clone)]
pub enum StackPreviewOption {
    AttachDebugger,
    Config(String),
    Diff,
    ExpectNoChanges,
    ImportFile(String),
    Message(String),
    Parallel(u32),
    Refresh(bool),
    Replace(String),
    ShowConfig,
    ShowReads,
    ShowReplacementSteps,
    ShowSames,
    ShowSecrets(bool),
    Target(String),
    TargetDependents,
    TargetReplace(String),
}

pub enum StackExportOption {
    File(String),
    ShowSecrets(bool),
}

pub enum StateDeleteOption {
    All,
    Force,
    TargetDependents,
}

pub enum StateMoveOption {
    IncludeParents,
}

pub enum ImportOption {
    Diff,
    File(String),
    From(String),
    Message(String),
    Out(String),
    Parallel(u32),
    Parent(String, String), // name=urn
    PreviewOnly,
    Properties(Vec<String>), // a,b,c,d...
    Protect(bool),
    Provider(String, String), // name=urn
}

pub enum StackListOption {
    All,
    Organization(String),
    Project(String),
    Tag(String, String), // tag-name=tag-value
}

#[derive(Debug, Clone, Deserialize)]
pub struct ProgramConfig {
    pub name: String,
}

impl ProgramConfig {
    pub fn from_file<P: AsRef<Path>>(path: P) -> Result<Self, Box<dyn std::error::Error>> {
        let content = std::fs::read_to_string(path)?;
        let config: ProgramConfig = serde_yaml::from_str(&content)?;
        Ok(config)
    }
}

#[derive(Debug, Clone, Deserialize)]
pub struct StackExport {
    pub deployment: StackDeployment,
}

#[derive(Debug, Clone, Deserialize)]
pub struct StackDeployment {
    pub resources: Option<Vec<StackResource>>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StackResource {
    pub urn: String,
    #[serde(rename = "type")]
    #[serde(skip_serializing_if = "Option::is_none")]
    pub type_: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub id: Option<String>,
    pub custom: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub provider: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dependencies: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub modified: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source_position: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub external: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub inputs: Option<serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub outputs: Option<serde_json::Value>,
}

#[derive(Deserialize, Serialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct StackPreview {
    pub config: serde_json::Value,
    pub steps: Vec<Step>,
    pub duration: u64,
    pub change_summary: Option<ChangeSummary>,
}

#[derive(Deserialize, Serialize, Debug, Clone, Display, Default, Copy)]
pub enum Operation {
    #[default]
    #[serde(rename = "same")]
    Same,
    #[serde(rename = "read")]
    Read,
    #[serde(rename = "create")]
    Create,
    #[serde(rename = "update")]
    Update,
    #[serde(rename = "delete")]
    Delete,
    #[serde(rename = "replace")]
    Replace,
    #[serde(rename = "import")]
    Import,
    #[serde(rename = "create-replacement")]
    CreateReplacement,
    #[serde(rename = "delete-replaced")]
    DeleteReplaced,
}

#[derive(Deserialize, Serialize, Debug, Clone, Default)]
#[serde(rename_all = "camelCase")]
pub struct Step {
    pub op: Operation,
    pub urn: String,
    pub provider: Option<String>,
    pub old_state: Option<StackResource>,
    pub new_state: Option<StackResource>,
    pub detailed_diff: Option<serde_json::Value>, // Using serde_json::Value for dynamic content
}

#[derive(Deserialize, Serialize, Debug, Clone)]
pub struct ChangeSummary {
    pub create: Option<u32>,
    pub delete: Option<u32>,
    pub same: Option<u32>,
    pub update: Option<u32>,
}

#[derive(Deserialize, Debug, Clone)]
struct LogEntry {
    sequence: u64,
    timestamp: u64,
    #[serde(flatten)]
    event: Event,
}

#[derive(Deserialize, Debug, Clone)]
#[serde(tag = "type", rename_all = "camelCase")]
enum Event {
    PreludeEvent {
        config: serde_json::Value,
    },
    DiagnosticEvent {
        prefix: String,
        message: String,
        color: String,
        severity: String,
    },
    ResourcePreEvent {
        metadata: ResourceMetadata,
    },
    ResOutputsEvent {
        metadata: ResourceMetadata,
    },
    SummaryEvent {
        maybe_corrupt: bool,
        duration_seconds: u64,
        resource_changes: ChangeSummary,
    },
    CancelEvent {},
}

#[derive(Deserialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
struct ResourceMetadata {
    op: String,
    urn: String,
    #[serde(rename = "type")]
    resource_type: String,
    old: Option<StackResource>,
    new: Option<StackResource>,
    detailed_diff: Option<serde_json::Value>,
    logical: bool,
    provider: String,
}

#[derive(Debug, Clone)]
pub struct Backend {
    pub url: String,
    pub extra_env_vars: Option<HashMap<String, String>>,
    pub auth_command: String, // TODO: command definition for things like `okta-aws-cli`
}

#[derive(Debug, Clone)]
pub struct CommandError {
    pub stdout: String,
    pub stderr: String,
    pub code: i32,
}

#[derive(Debug)]
pub enum PulumiExecutionError {
    CommandError(CommandError),
    JsonError(serde_json::Error),
    Utf8Error(std::string::FromUtf8Error),
    IoError(std::io::Error),
}

#[derive(Debug, Clone)]
pub struct LocalProgram {
    pub name: String,
    pub secrets_provider: Option<String>,
    pub cwd: String,
    pub extra_env_vars: Option<HashMap<String, String>>,
    pub backend: Option<Backend>,
}

impl LocalProgram {
    pub fn new(name: String, cwd: String) -> Self {
        LocalProgram {
            name,
            secrets_provider: None,
            cwd,
            extra_env_vars: None,
            backend: None,
        }
    }

    /// Returns a list of all local programs in the current working directory and its subdirectories up to the specified depth. looking for `Pulumi.yaml` files.
    pub fn all_in_cwd<P: AsRef<Path>>(
        cwd: P,
        depth: u32,
    ) -> Result<Vec<Self>, Box<dyn std::error::Error>> {
        let mut programs = Vec::new();

        for entry in std::fs::read_dir(&cwd)? {
            let entry = entry?;
            let path = entry.path();

            if path.is_dir() {
                if depth > 0 {
                    let sub_programs =
                        LocalProgram::all_in_cwd(path.to_string_lossy().to_string(), depth - 1)?;
                    programs.extend(sub_programs);
                }
            } else if path.ends_with("Pulumi.yaml") {
                programs.push(Self::from_cwd(cwd.as_ref())?);
            }
        }

        Ok(programs)
    }

    pub fn from_cwd<P: AsRef<Path>>(cwd: P) -> Result<Self, Box<dyn std::error::Error>> {
        let config = ProgramConfig::from_file(cwd.as_ref().join("Pulumi.yaml"))?;

        Ok(LocalProgram {
            name: config.name,
            secrets_provider: None,
            cwd: cwd.as_ref().to_string_lossy().to_string(),
            extra_env_vars: None,
            backend: None,
        })
    }

    pub fn with_secrets_provider(mut self, secrets_provider: String) -> Self {
        self.secrets_provider = Some(secrets_provider);
        self
    }

    pub fn with_extra_env_vars(mut self, extra_env_vars: HashMap<String, String>) -> Self {
        self.extra_env_vars = Some(extra_env_vars);
        self
    }

    pub fn with_backend(mut self, backend: Backend) -> Self {
        self.backend = Some(backend);
        self
    }

    fn run_command<R>(&self, args: Vec<&str>) -> Result<(R, String, i32), PulumiExecutionError>
    where
        R: serde::de::DeserializeOwned,
    {
        let mut env_vars: HashMap<String, String> =
            self.extra_env_vars.clone().unwrap_or(HashMap::new());

        if let Some(backend) = &self.backend {
            if let Some(extra_env_vars) = &backend.extra_env_vars {
                for (key, value) in extra_env_vars {
                    env_vars.insert(key.to_string(), value.to_string());
                }
            }
        }

        let (stdout, stderr, code) = self.run_pulumi_command(env_vars, args)?;

        if code != 0 {
            return Err(PulumiExecutionError::CommandError(CommandError {
                stdout,
                stderr,
                code,
            }));
        }

        let result: R = serde_json::from_str(&stdout).map_err(PulumiExecutionError::JsonError)?;

        Ok((result, stdout, code))
    }

    fn run_pulumi_command(
        &self,
        env_vars: HashMap<String, String>,
        args: Vec<&str>,
    ) -> Result<(String, String, i32), PulumiExecutionError> {
        let output = std::process::Command::new("pulumi")
            .args(args)
            .args(&["--cwd", &self.cwd])
            .envs(env_vars)
            .output()
            .map_err(PulumiExecutionError::IoError)?;

        let stdout = String::from_utf8(output.stdout).map_err(PulumiExecutionError::Utf8Error)?;
        let stderr = String::from_utf8(output.stderr).map_err(PulumiExecutionError::Utf8Error)?;

        Ok((stdout, stderr, output.status.code().unwrap()))
    }
}

impl Interactor for LocalProgram {
    type Error = PulumiExecutionError;

    fn stack_list(&self, options: Vec<StackListOption>) -> Result<Vec<Stack>, Self::Error> {
        let mut args = vec!["stack", "ls", "--json"];

        let additional_options = options.iter().flat_map(|option| match option {
            StackListOption::All => vec!["--all"],
            StackListOption::Organization(org) => vec!["--organization", org],
            StackListOption::Project(proj) => vec!["--project", proj],
            _ => todo!(),
        });

        args.extend(additional_options);

        let (result, _, _) = self.run_command::<Vec<Stack>>(args)?;

        Ok(result)
    }

    fn stack_export(
        &self,
        stack_name: &str,
        options: Vec<StackExportOption>,
    ) -> Result<StackExport, Self::Error> {
        let mut args = vec!["stack", "export", "--stack", stack_name];

        let additional_options = options.iter().flat_map(|option| match option {
            StackExportOption::File(file) => vec!["--file", file],
            StackExportOption::ShowSecrets(show) => {
                if *show {
                    vec!["--show-secrets"]
                } else {
                    vec![]
                }
            }
        });

        args.extend(additional_options);

        let (result, _, _) = self.run_command::<StackExport>(args)?;

        Ok(result)
    }

    fn stack_output(
        &self,
        stack_name: &str,
        options: Vec<StackOutputOption>,
    ) -> Result<serde_json::Value, Self::Error> {
        let mut args = vec!["stack", "output", "--stack", stack_name, "--json"];

        let additional_options = options.iter().flat_map(|option| match option {
            StackOutputOption::ShowSecrets(show) => {
                if *show {
                    vec!["--show-secrets"]
                } else {
                    vec![]
                }
            }
        });

        args.extend(additional_options);

        let (result, _, _) = self.run_command::<serde_json::Value>(args)?;

        Ok(result)
    }

    fn stack_preview(
        &self,
        stack_name: &str,
        options: Vec<StackPreviewOption>,
    ) -> Result<StackPreview, Self::Error> {
        let mut args = vec!["preview", "--stack", stack_name, "--json"];

        let additional_options = options.iter().flat_map(|option| match option {
            StackPreviewOption::Diff => vec!["--diff"],
            StackPreviewOption::ExpectNoChanges => vec!["--expect-no-changes"],
            StackPreviewOption::Message(msg) => vec!["--message", msg],
            StackPreviewOption::Refresh(refresh) => {
                if *refresh {
                    vec!["--refresh"]
                } else {
                    vec![]
                }
            }
            StackPreviewOption::Replace(replace) => vec!["--replace", replace],
            StackPreviewOption::ShowConfig => vec!["--show-config"],
            StackPreviewOption::ShowReads => vec!["--show-reads"],
            StackPreviewOption::ShowReplacementSteps => vec!["--show-replacement-steps"],
            StackPreviewOption::ShowSames => vec!["--show-sames"],
            StackPreviewOption::ShowSecrets(show) => {
                if *show {
                    vec!["--show-secrets"]
                } else {
                    vec![]
                }
            }
            StackPreviewOption::Target(target) => vec!["--target", target],
            StackPreviewOption::TargetDependents => vec!["--target-dependents"],
            StackPreviewOption::TargetReplace(target_replace) => {
                vec!["--target-replace", target_replace]
            }
            _ => todo!(),
        });

        args.extend(additional_options);

        let (result, _, _) = self.run_command::<StackPreview>(args)?;

        Ok(result)
    }
}
