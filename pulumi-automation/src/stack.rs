#![allow(dead_code)]

use std::collections::HashMap;

use serde::{Deserialize, Serialize};
use tokio::sync::mpsc::Sender;

use crate::event::EngineEvent;

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ResourceState {
    pub inputs: Option<serde_json::Value>,
    pub outputs: Option<serde_json::Value>,
    pub urn: String,
    #[serde(rename = "type")]
    pub _type: String,
    pub id: Option<String>,
    pub provider: Option<String>,
    pub parent: Option<String>,
    pub source_position: Option<String>,
    pub modified: Option<String>,
    pub custom: Option<bool>,
    pub external: Option<bool>,
    pub created: Option<String>,
    pub dependencies: Option<Vec<String>>,
    pub additional_secret_outputs: Option<Vec<String>>,
    pub property_dependencies: Option<HashMap<String, Vec<String>>>,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize, Eq, PartialEq, Hash)]
#[serde(rename_all = "kebab-case")]
pub enum Operation {
    Same,
    Read,
    Create,
    Update,
    Delete,
    Replace,
    Refresh,
    Import,
    CreateReplacement,
    DeleteReplaced,
    ReadReplacement,
    Discard,
    DiscardReplaced,
    RemovePendingReplace,
    ImportReplacement,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StackChangeStep {
    pub op: Operation,
    pub urn: String,
    pub provider: Option<String>,
    pub new_state: Option<ResourceState>,
    pub old_state: Option<ResourceState>,
    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StackChangeSummary {
    pub steps: Vec<StackChangeStep>,
    pub change_summary: Option<HashMap<Operation, usize>>,
    pub duration: i64,
    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

pub struct PulumiProcessListener {
    pub preview_tx: Option<Sender<StackChangeSummary>>,
    pub event_tx: Sender<EngineEvent>,
}

#[derive(Debug, Clone, Default)]
pub struct StackRefreshOptions {
    pub skip_preview: Option<bool>,
    pub target: Option<Vec<String>>,
    pub target_dependents: Option<bool>,
    pub exclude: Option<Vec<String>>,
    pub exclude_dependents: Option<bool>,
    pub show_secrets: Option<bool>,
    pub continue_on_error: Option<bool>,
}

#[derive(Debug, Clone, Default)]
pub struct StackDestroyOptions {
    pub skip_preview: Option<bool>,
    pub refresh: Option<bool>,
    pub target: Option<Vec<String>>,
    pub target_dependents: Option<bool>,
    pub exclude: Option<Vec<String>>,
    pub exclude_dependents: Option<bool>,
    pub show_secrets: Option<bool>,
    pub continue_on_error: Option<bool>,
    pub exclude_protected: Option<bool>,
}

#[derive(Debug, Clone, Default)]
pub struct StackUpOptions {
    pub skip_preview: Option<bool>,
    pub preview: Option<bool>,
    pub refresh: Option<bool>,
    pub replace: Option<Vec<String>>,
    pub diff: Option<bool>,
    pub target: Option<Vec<String>>,
    pub target_dependents: Option<bool>,
    pub exclude: Option<Vec<String>>,
    pub exclude_dependents: Option<bool>,
    pub show_secrets: Option<bool>,
    pub continue_on_error: Option<bool>,
    pub expect_no_changes: Option<bool>,
    pub show_reads: Option<bool>,
    pub show_replacement_steps: Option<bool>,
}

#[derive(Debug, Clone, Default)]
pub struct StackPreviewOptions {
    pub refresh: Option<bool>,
    pub replace: Option<Vec<String>>,
    pub diff: Option<bool>,
    pub target: Option<Vec<String>>,
    pub target_dependents: Option<bool>,
    pub exclude: Option<Vec<String>>,
    pub exclude_dependents: Option<bool>,
    pub show_secrets: Option<bool>,
    pub continue_on_error: Option<bool>,
    pub expect_no_changes: Option<bool>,
    pub show_reads: Option<bool>,
    pub show_replacement_steps: Option<bool>,
}

#[async_trait::async_trait]
pub trait Stack: Sized {
    fn preview(&self, options: StackPreviewOptions) -> super::Result<StackChangeSummary>;
    async fn up(
        &self,
        options: StackUpOptions,
        listener: PulumiProcessListener,
    ) -> super::Result<()>;
    async fn refresh(
        &self,
        options: StackRefreshOptions,
        listener: PulumiProcessListener,
    ) -> super::Result<()>;
    async fn destroy(
        &self,
        options: StackDestroyOptions,
        listener: PulumiProcessListener,
    ) -> super::Result<()>;
}
