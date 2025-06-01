use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Default, Deserialize, Serialize)]
pub struct ConfigValue {
    pub value: String,
    pub secret: bool,
}

#[derive(Debug, Clone, Default, Deserialize, Serialize)]
pub struct StackSettings {
    pub secrets_provider: Option<String>,
    pub encrypted_key: Option<String>,
    pub encryption_salt: Option<String>,
    pub config: serde_json::Value,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct TokenInformation {
    pub name: String,
    pub organization: Option<String>,
    pub team: Option<String>,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct Whoami {
    pub user: String,
    pub url: Option<String>,
    pub organizations: Option<Vec<String>>,
    pub token_information: Option<TokenInformation>,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct StackRemoveOptions {
    pub force: Option<bool>,
    pub preserve_config: Option<bool>,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct StackListOptions {
    pub all: Option<bool>,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct StackSummary {
    pub name: String,
    pub current: bool,
    pub last_update: Option<String>,
    pub update_in_progress: Option<bool>,
    pub resource_count: Option<usize>,
    pub url: Option<String>,
}

#[derive(Debug, Clone, Default, Deserialize, Serialize)]
pub struct Deployment {
    pub version: usize,
    pub deployment: serde_json::Value,
}

#[derive(Debug, Clone, Default)]
pub struct StackCreateOptions {
    pub secrets_provider: Option<String>,
    pub copy_config_from: Option<String>,
}

pub type ConfigMap = std::collections::HashMap<String, ConfigValue>;
pub type OutputMap = std::collections::HashMap<String, serde_json::Value>;

pub trait Workspace {
    type Stack: super::stack::Stack;

    fn whoami(&self) -> super::Result<Whoami>;
    fn get_config(&self, stack_name: &str, key: &str, path: bool) -> super::Result<ConfigValue>;
    fn set_config(
        &self,
        stack_name: &str,
        key: &str,
        path: bool,
        value: ConfigValue,
    ) -> super::Result<()>;
    fn remove_config(&self, stack_name: &str, key: &str, path: bool) -> super::Result<()>;
    fn create_stack(&self, name: &str, options: StackCreateOptions) -> super::Result<Self::Stack>;
    fn select_stack(&self, name: &str) -> super::Result<Self::Stack>;
    fn select_or_create_stack(
        &self,
        name: &str,
        options: Option<StackCreateOptions>,
    ) -> super::Result<Self::Stack>;
    fn remove_stack(
        &self,
        stack_name: &str,
        options: Option<StackRemoveOptions>,
    ) -> super::Result<()>;
    fn list_stacks(&self, options: Option<StackListOptions>) -> super::Result<Vec<StackSummary>>;
    fn export_stack(&self, stack_name: &str) -> super::Result<Deployment>;
    fn import_stack(&self, stack_name: &str, deployment: Deployment) -> super::Result<()>;
    fn stack_outputs(&self, stack_name: &str) -> super::Result<OutputMap>;
}
