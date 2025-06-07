use std::collections::HashMap;

use serde::{Deserialize, Serialize};

use crate::stack::{Operation, ResourceType};

/// A Pulumi engine event, such as a change to a resource or diagnostic message.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EngineEvent {
    pub sequence: Option<i64>,
    pub timestamp: Option<i64>,
    #[serde(flatten)]
    pub event: EventType,

    #[serde(flatten)]
    extra: serde_json::Value,
}

/// The type of event that occurred in the Pulumi engine.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(untagged)]
pub enum EventType {
    CancelEvent {
        #[serde(rename = "cancelEvent")]
        details: CancelDetails,
    },
    Diagnostic {
        #[serde(rename = "diagnosticEvent")]
        details: DiagnosticEventDetails,
    },
    PreludeEvent {
        #[serde(rename = "preludeEvent")]
        details: PreludeDetails,
    },
    SummaryEvent {
        #[serde(rename = "summaryEvent")]
        details: SummaryDetails,
    },
    ResourcePreEvent {
        #[serde(rename = "resourcePreEvent")]
        details: ResourcePreDetails,
    },
    ResOutputsEvent {
        #[serde(rename = "resOutputsEvent")]
        details: ResOutputsDetails,
    },
    StdoutEvent {
        #[serde(rename = "stdoutEvent")]
        details: StdoutDetails,
    },
    ResOpFailedEvent {
        #[serde(rename = "resOpFailedEvent")]
        details: ResOpFailedDetails,
    },
    PolicyEvent {
        #[serde(rename = "policyEvent")]
        details: PolicyDetails,
    },
    StartDebuggingEvent {
        #[serde(rename = "startDebuggingEvent")]
        details: StartDebuggingDetails,
    },
    ProgressEvent {
        #[serde(rename = "progressEvent")]
        details: ProgressDetails,
    },
    Unknown {
        #[serde(flatten)]
        extra: serde_json::Value,
    },
}

impl EventType {
    pub fn to_string(&self) -> String {
        match self {
            EventType::CancelEvent { .. } => "cancelEvent".to_string(),
            EventType::Diagnostic { .. } => "diagnosticEvent".to_string(),
            EventType::PreludeEvent { .. } => "preludeEvent".to_string(),
            EventType::SummaryEvent { .. } => "summaryEvent".to_string(),
            EventType::ResourcePreEvent { .. } => "resourcePreEvent".to_string(),
            EventType::ResOutputsEvent { .. } => "resOutputsEvent".to_string(),
            EventType::StdoutEvent { .. } => "stdoutEvent".to_string(),
            EventType::ResOpFailedEvent { .. } => "resOpFailedEvent".to_string(),
            EventType::PolicyEvent { .. } => "policyEvent".to_string(),
            EventType::StartDebuggingEvent { .. } => "startDebuggingEvent".to_string(),
            EventType::ProgressEvent { .. } => "progressEvent".to_string(),
            EventType::Unknown { .. } => "unknown".to_string(),
        }
    }
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "kebab-case")]
pub enum DiagnosticSeverity {
    Debug,
    Info,
    #[serde(rename = "info#err")]
    InfoErr,
    Warning,
    Error,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct DiagnosticEventDetails {
    pub urn: Option<String>,
    pub prefix: Option<String>,
    pub message: String,
    pub severity: DiagnosticSeverity,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PreludeDetails {
    pub config: HashMap<String, String>,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct StepEventStateMetadata {
    pub inputs: HashMap<String, serde_json::Value>,
    pub outputs: HashMap<String, serde_json::Value>,
    pub urn: String,
    #[serde(rename = "type")]
    pub resource_type: String,
    pub id: String,
    pub provider: String,
    pub parent: String,
    pub custom: Option<bool>,
    pub delete: Option<bool>,
    pub protect: Option<bool>,
    pub retain_on_delete: Option<bool>,
    pub init_errors: Option<Vec<String>>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "kebab-case")]
pub enum DiffKind {
    Add,
    AddReplace,
    Delete,
    DeleteReplace,
    Update,
    UpdateReplace,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PropertyDiff {
    pub diff_kind: DiffKind,
    pub input_diff: bool,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct StepEventMetadata {
    pub op: Operation,
    pub urn: String,
    #[serde(rename = "type")]
    pub resource_type: ResourceType,
    pub old: Option<StepEventStateMetadata>,
    pub new: Option<StepEventStateMetadata>,
    pub keys: Option<Vec<String>>,
    pub diffs: Option<Vec<String>>,
    pub detailed_diff: Option<HashMap<String, PropertyDiff>>,
    pub logical: Option<bool>,
    pub provider: String,
}

impl StepEventMetadata {
    pub fn name(&self) -> String {
        self.urn
            .split("::")
            .last()
            .map(|s| s.to_string())
            .unwrap_or_else(|| self.urn.clone())
    }
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ResourcePreDetails {
    pub metadata: StepEventMetadata,
    pub planning: Option<bool>,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ResOutputsDetails {
    pub metadata: StepEventMetadata,
    pub planning: Option<bool>,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct CancelDetails {
    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SummaryDetails {
    pub duration_seconds: i64,
    pub resource_changes: HashMap<Operation, i64>,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct StdoutDetails {
    pub message: String,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ResOpFailedDetails {
    pub metadata: StepEventMetadata,
    pub status: i64,
    pub steps: i64,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub enum EnforcementLevel {
    Warning,
    Mandatory,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PolicyDetails {
    pub resource_urn: Option<String>,
    pub message: String,
    pub policy_name: String,
    pub policy_pack_name: String,
    pub policy_pack_version: String,
    pub policy_pack_version_tag: String,
    pub enforcement_level: EnforcementLevel,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct StartDebuggingDetails {
    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ProgressDetails {
    pub done: bool,
    pub id: String,
    pub message: String,
    pub received: i64,
    pub total: i64,
    #[serde(rename = "type")]
    pub _type: String,

    #[serde(flatten)]
    pub extra_values: Option<serde_json::Value>,
}
