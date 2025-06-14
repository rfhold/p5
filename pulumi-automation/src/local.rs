#![allow(unused)]

use std::collections::HashMap;
use std::io::Write;
use std::process::Stdio;
use tokio::{
    io::{AsyncBufReadExt, BufReader},
    sync::mpsc::Sender,
};
use tracing::instrument;

use crate::{stack::Stack, workspace::Workspace};

#[derive(Debug, Clone, Default)]
pub struct LocalWorkspace {
    pub cwd: String,
}

impl LocalWorkspace {
    pub fn new(cwd: String) -> Self {
        Self { cwd }
    }

    #[instrument(
        skip(self),
        fields(
            cwd = %self.cwd,
        ),
    )]
    fn run_command_sync(
        &self,
        args: Vec<&str>,
    ) -> Result<(Vec<u8>, Vec<u8>, Option<i32>), super::PulumiError> {
        let env_vars: HashMap<String, String> = HashMap::new();

        let output = std::process::Command::new("pulumi")
            .args(args)
            .args(&["--cwd", &self.cwd])
            .envs(env_vars)
            .output()
            .map_err(|e| super::PulumiError::Other(format!("Failed to execute command: {}", e)))?;

        let stdout = output.stdout;
        let stderr = output.stderr;
        let code = output.status.code();

        if let Some(exit_code) = code {
            if exit_code != 0 {
                return Err(super::PulumiError::Other(format!(
                    "Command failed with exit code {}: stderr: {}. Output: {}",
                    exit_code,
                    String::from_utf8_lossy(&stderr),
                    String::from_utf8_lossy(&stdout),
                )));
            }
        }

        Ok((stdout, stderr, code))
    }

    fn run_command_sync_output<O>(
        &self,
        args: Vec<&str>,
    ) -> Result<(O, Vec<u8>, Option<i32>), super::PulumiError>
    where
        O: serde::de::DeserializeOwned,
    {
        let (stdout, stderr, code) = self.run_command_sync(args)?;

        let output: O = serde_json::from_slice(&stdout)
            .map_err(|e| super::PulumiError::Other(format!("Failed to parse output: {}", e)))?;

        Ok((output, stderr, code))
    }

    #[instrument(
        name = "run_command_async",
        skip(self),
        fields(
            cwd = %self.cwd,
        ),
    )]
    async fn run_command_piped<E>(
        &self,
        args: Vec<&str>,
        output_tx: Sender<E>,
    ) -> Result<(), super::PulumiError>
    where
        E: serde::de::DeserializeOwned + Send + 'static,
    {
        let env_vars: HashMap<String, String> = HashMap::new();

        let mut child = tokio::process::Command::new("pulumi")
            .args(args)
            .args(&["--cwd", &self.cwd])
            .envs(env_vars)
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()
            .map_err(|e| super::PulumiError::Other(format!("Failed to spawn command: {}", e)))?;

        let stdout = child.stdout.take().ok_or_else(|| {
            super::PulumiError::Other("Failed to take stdout from child process".to_string())
        })?;

        let stderr = child.stderr.take().ok_or_else(|| {
            super::PulumiError::Other("Failed to take stderr from child process".to_string())
        })?;

        let stdout_lines = BufReader::new(stdout).lines();
        let mut stderr_reader = BufReader::new(stderr).lines();

        let mut json_rx = crate::json_reader::parse_json_stream(stdout_lines);

        let process = tokio::spawn(async move {
            child.wait().await.expect("Child process failed to finish");
        });

        let reader1 = tokio::spawn(async move {
            while let Some(result) = json_rx.recv().await {
                match result {
                    Ok(event) => {
                        if let Err(e) = output_tx.send(event).await {
                            eprintln!("Failed to send event: {}", e);
                            break;
                        }
                    }
                    Err(e) => {
                        eprintln!("JSON parsing error: {}", e);
                    }
                }
            }
        });

        let reader2 = tokio::spawn(async move {
            while let Some(line) = stderr_reader.next_line().await.unwrap() {
                eprintln!("stderr: {}", line);
            }
        });

        tokio::try_join!(process, reader1, reader2).map_err(|e| {
            super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
        })?;

        Ok(())
    }

    async fn run_engine_operation(
        &self,
        args: Vec<&str>,
        event_tx: Sender<crate::event::EngineEvent>,
        preview_tx: Option<Sender<crate::stack::StackChangeSummary>>,
    ) -> Result<(), super::PulumiError> {
        let (output_tx, mut output_rx) = tokio::sync::mpsc::channel::<serde_json::Value>(100);

        let should_capture_preview = !args.contains(&"--skip-preview") && preview_tx.is_some();

        let listener = tokio::spawn(async move {
            let mut try_capture_preview = should_capture_preview;
            while let Some(value) = output_rx.recv().await {
                if try_capture_preview {
                    if let Some(preview_tx) = &preview_tx {
                        match serde_json::from_value::<crate::stack::StackChangeSummary>(
                            value.clone(),
                        ) {
                            Ok(summary) => {
                                // TODO: Strict checking/ piping uncaptured values
                                if preview_tx.send(summary).await.is_err() {
                                    eprintln!("Failed to send preview summary, channel closed");
                                    break;
                                }
                            }
                            Err(e) => {
                                eprintln!("Failed to deserialize preview summary: {}", e);
                            }
                        }
                    }
                    try_capture_preview = false; // Only capture the first output
                    continue; // Skip further processing for this value
                }
                match serde_json::from_value::<crate::event::EngineEvent>(value.clone()) {
                    Ok(event) => {
                        if event_tx.send(event).await.is_err() {
                            eprintln!("Failed to send event, channel closed");
                            break;
                        }
                    }
                    Err(e) => {
                        eprintln!("Failed to deserialize event: {}, {:?}", e, value);
                    }
                }
            }
        });

        self.run_command_piped(args, output_tx).await.map_err(|e| {
            super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
        })?;

        listener
            .await
            .map_err(|e| super::PulumiError::Other(format!("Listener task failed: {:?}", e)))?;

        Ok(())
    }

    fn get_stack_file_path(&self, stack_name: &str) -> Option<String> {
        let extensions = vec!["yaml", "yml"];

        for ext in extensions {
            let path = format!("{}/Pulumi.{}.{}", self.cwd, stack_name, ext);
            if std::path::Path::new(&path).exists() {
                return Some(path);
            }
        }

        None
    }
}

impl Workspace for LocalWorkspace {
    type Stack = LocalStack;

    #[instrument(skip(self))]
    fn whoami(&self) -> crate::Result<crate::workspace::Whoami> {
        let (stdout, stderr, code) = self
            .run_command_sync_output(vec!["whoami", "--json"])
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        return Ok(stdout);
    }

    #[instrument(skip(self))]
    fn create_stack(
        &self,
        name: &str,
        options: crate::workspace::StackCreateOptions,
    ) -> crate::Result<Self::Stack> {
        let mut args = vec!["stack", "init", name];

        if let Some(provider) = options.secrets_provider.as_ref() {
            args.push("--secrets-provider");
            args.push(&provider);
        }

        if let Some(copy_from) = options.copy_config_from.as_ref() {
            args.push("--copy-config-from");
            args.push(&copy_from);
        }

        let (stdout, stderr, code) = self.run_command_sync(args).map_err(|e| {
            super::PulumiError::Other(format!("Failed to run local command: {:?}", e))
        })?;

        return Ok(LocalStack {
            name: name.to_string(),
            workspace: self.clone(),
        });
    }

    #[instrument(skip(self))]
    fn select_stack(&self, name: &str) -> crate::Result<Self::Stack> {
        let (stdout, stderr, code) = self
            .run_command_sync(vec!["stack", "select", name])
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        return Ok(LocalStack {
            name: name.to_string(),
            workspace: self.clone(),
        });
    }

    #[instrument(skip(self))]
    fn select_or_create_stack(
        &self,
        name: &str,
        options: Option<crate::workspace::StackCreateOptions>,
    ) -> crate::Result<Self::Stack> {
        let stack = self.select_stack(name);
        if stack.is_ok() {
            return stack;
        }
        let create_options = options.unwrap_or_default();

        // The copy_config_from option is not able to be implemented in the select with --create
        self.create_stack(name, create_options).map_err(|e| {
            super::PulumiError::Other(format!("Failed to create or select stack: {:?}", e))
        })
    }

    #[instrument(skip(self))]
    fn remove_stack(
        &self,
        stack_name: &str,
        options: Option<crate::workspace::StackRemoveOptions>,
    ) -> crate::Result<()> {
        let mut args = vec!["stack", "rm", stack_name, "--yes"];

        if let Some(opts) = options {
            if opts.force.unwrap_or(false) {
                args.push("--force");
            }
            if opts.preserve_config.unwrap_or(false) {
                args.push("--preserve-config");
            }
        }

        let (stdout, stderr, code) = self.run_command_sync(args).map_err(|e| {
            super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
        })?;

        Ok(())
    }

    #[instrument(skip(self))]
    fn list_stacks(
        &self,
        options: Option<crate::workspace::StackListOptions>,
    ) -> crate::Result<Vec<crate::workspace::StackSummary>> {
        let mut args = vec!["stack", "ls", "--json"];

        if let Some(opts) = options {
            if opts.all.unwrap_or(false) {
                args.push("--all");
            }
        }

        let (stdout, stderr, code) = self
            .run_command_sync_output::<Vec<crate::workspace::StackSummary>>(args)
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        Ok(stdout)
    }

    #[instrument(skip(self))]
    fn export_stack(&self, stack_name: &str) -> crate::Result<crate::workspace::Deployment> {
        let args = vec!["stack", "export", "--stack", stack_name];
        let (stdout, stderr, code) = self
            .run_command_sync_output::<crate::workspace::Deployment>(args)
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        return Ok(stdout);
    }

    #[instrument(skip(self))]
    fn import_stack(
        &self,
        stack_name: &str,
        deployment: crate::workspace::Deployment,
    ) -> crate::Result<()> {
        let deployment_json = serde_json::to_string(&deployment).map_err(|e| {
            super::PulumiError::Other(format!("Failed to serialize deployment: {}", e))
        })?;
        let mut temp_file = tempfile::NamedTempFile::new().map_err(|e| {
            super::PulumiError::Other(format!("Failed to create temporary file: {}", e))
        })?;

        write!(temp_file, "{}", deployment_json).map_err(|e| {
            super::PulumiError::Other(format!("Failed to write to temporary file: {}", e))
        })?;

        let args = vec![
            "stack",
            "import",
            "--stack",
            stack_name,
            "--file",
            temp_file.path().to_str().unwrap(),
        ];

        let (stdout, stderr, code) = self.run_command_sync(args).map_err(|e| {
            super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
        })?;

        return Ok(());
    }

    #[instrument(skip(self))]
    fn stack_outputs(&self, stack_name: &str) -> crate::Result<crate::workspace::OutputMap> {
        let args = vec![
            "stack",
            "output",
            "--json",
            "--stack",
            stack_name,
            "--show-secrets",
        ];
        let (stdout, stderr, code) = self
            .run_command_sync_output::<crate::workspace::OutputMap>(args)
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        return Ok(stdout);
    }

    #[instrument(skip(self))]
    fn get_config(
        &self,
        stack_name: &str,
        key: &str,
        path: bool,
    ) -> crate::Result<crate::workspace::ConfigValue> {
        let mut args = vec!["config", "get", "--json", "--stack", stack_name];
        if path {
            args.push("--path");
        }
        args.push(key);

        let (stdout, stderr, code) = self
            .run_command_sync_output::<crate::workspace::ConfigValue>(args)
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        return Ok(stdout);
    }

    #[instrument(skip(self))]
    fn set_config(
        &self,
        stack_name: &str,
        key: &str,
        path: bool,
        value: crate::workspace::ConfigValue,
    ) -> crate::Result<()> {
        let mut args = vec!["config", "set"];
        if path {
            args.push("--path");
        }
        args.push(key);
        args.push("--stack");
        args.push(stack_name);
        args.push(if value.secret {
            "--secret"
        } else {
            "--plaintext"
        });
        args.push("--non-interactive");
        args.push(&value.value);

        let (stdout, stderr, code) = self.run_command_sync(args).map_err(|e| {
            super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
        })?;

        Ok(())
    }

    #[instrument(skip(self))]
    fn remove_config(&self, stack_name: &str, key: &str, path: bool) -> crate::Result<()> {
        let mut args = vec!["config", "rm", key, "--stack", stack_name];
        if path {
            args.push("--path");
        }

        let (stdout, stderr, code) = self.run_command_sync(args).map_err(|e| {
            super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
        })?;

        Ok(())
    }

    fn get_stack_config(&self, stack_name: &str) -> crate::Result<crate::workspace::StackSettings> {
        if let Some(path) = self.get_stack_file_path(stack_name) {
            let content = std::fs::read_to_string(path).map_err(|e| {
                super::PulumiError::Other(format!("Failed to read stack file: {}", e))
            })?;
            let settings: crate::workspace::StackSettings = serde_yaml::from_str(&content)
                .map_err(|e| {
                    super::PulumiError::Other(format!("Failed to parse stack settings: {}", e))
                })?;
            return Ok(settings);
        }

        Err(super::PulumiError::Other(
            "Stack file not found".to_string(),
        ))
    }

    fn set_stack_config(
        &self,
        stack_name: &str,
        config: crate::workspace::StackSettings,
    ) -> crate::Result<()> {
        if let Some(path) = self.get_stack_file_path(stack_name) {
            let content = serde_yaml::to_string(&config).map_err(|e| {
                super::PulumiError::Other(format!("Failed to serialize stack settings: {}", e))
            })?;
            std::fs::write(path, content).map_err(|e| {
                super::PulumiError::Other(format!("Failed to write stack file: {}", e))
            })?;
            return Ok(());
        }
        Err(super::PulumiError::Other(
            "Stack file not found".to_string(),
        ))
    }
}

#[derive(Debug, Clone)]
pub struct LocalStack {
    pub name: String,
    pub workspace: LocalWorkspace,
}

#[async_trait::async_trait]
impl Stack for LocalStack {
    #[instrument(skip(self))]
    fn preview(
        &self,
        options: crate::stack::StackPreviewOptions,
    ) -> crate::Result<crate::stack::StackChangeSummary> {
        let mut args = vec![
            "preview",
            "--stack",
            &self.name,
            "--non-interactive",
            "--json",
            "--diff",
        ];

        if let Some(refresh) = options.refresh {
            if refresh {
                args.push("--refresh");
            }
        }

        if let Some(replace) = options.replace.as_ref() {
            for r in replace {
                args.push("--replace");
                args.push(&r);
            }
        }

        if let Some(diff) = options.diff {
            if diff {
                args.push("--diff");
            }
        }

        if let Some(target) = options.target.as_ref() {
            for t in target {
                args.push("--target");
                args.push(&t);
            }
        }

        if let Some(target_dependents) = options.target_dependents {
            if target_dependents {
                args.push("--target-dependents");
            }
        }

        if let Some(exclude) = options.exclude.as_ref() {
            for e in exclude {
                args.push("--exclude");
                args.push(&e);
            }
        }

        if let Some(exclude_dependents) = options.exclude_dependents {
            if exclude_dependents {
                args.push("--exclude-dependents");
            }
        }

        if let Some(show_secrets) = options.show_secrets {
            if show_secrets {
                args.push("--show-secrets");
            }
        }

        if let Some(continue_on_error) = options.continue_on_error {
            if continue_on_error {
                args.push("--continue-on-error");
            }
        }

        if let Some(expect_no_changes) = options.expect_no_changes {
            if expect_no_changes {
                args.push("--expect-no-changes");
            }
        }

        if let Some(show_reads) = options.show_reads {
            if show_reads {
                args.push("--show-reads");
            }
        }

        if let Some(show_replacement_steps) = options.show_replacement_steps {
            if show_replacement_steps {
                args.push("--show-replacement-steps");
            }
        }

        let (summary, _, _) = self
            .workspace
            .run_command_sync_output::<crate::stack::StackChangeSummary>(args)
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        Ok(summary)
    }

    async fn preview_async(
        &self,
        options: crate::stack::StackPreviewOptions,
        listener: crate::stack::PulumiProcessListener,
    ) -> crate::Result<()> {
        if let Some(preview_tx) = listener.preview_tx {
            if let Ok(summary) = self.preview(options) {
                preview_tx.send(summary).await.map_err(|e| {
                    super::PulumiError::Other(format!("Failed to send preview summary: {:?}", e))
                })?;

                return Ok(());
            } else {
                return Err(super::PulumiError::Other(
                    "Failed to get preview summary".to_string(),
                ));
            }
        } else {
            return Err(super::PulumiError::Other(
                "Preview channel not available".to_string(),
            ));
        }
    }

    #[instrument(skip(self))]
    async fn up(
        &self,
        options: crate::stack::StackUpOptions,
        listener: crate::stack::PulumiProcessListener,
    ) -> crate::Result<()> {
        let mut args = vec![
            "up",
            "--stack",
            &self.name,
            "--non-interactive",
            "--json",
            "--diff",
        ];

        if let Some(skip_preview) = options.skip_preview {
            if skip_preview {
                args.push("--skip-preview");
            }
        } else {
            args.push("--yes");
        }

        if let Some(refresh) = options.refresh {
            if refresh {
                args.push("--refresh");
            }
        }

        if let Some(replace) = options.replace.as_ref() {
            for r in replace {
                args.push("--replace");
                args.push(&r);
            }
        }

        if let Some(diff) = options.diff {
            if diff {
                args.push("--diff");
            }
        }

        if let Some(target) = options.target.as_ref() {
            for t in target {
                args.push("--target");
                args.push(&t);
            }
        }

        if let Some(target_dependents) = options.target_dependents {
            if target_dependents {
                args.push("--target-dependents");
            }
        }

        if let Some(exclude) = options.exclude.as_ref() {
            for e in exclude {
                args.push("--exclude");
                args.push(&e);
            }
        }

        if let Some(exclude_dependents) = options.exclude_dependents {
            if exclude_dependents {
                args.push("--exclude-dependents");
            }
        }

        if let Some(show_secrets) = options.show_secrets {
            if show_secrets {
                args.push("--show-secrets");
            }
        }

        if let Some(continue_on_error) = options.continue_on_error {
            if continue_on_error {
                args.push("--continue-on-error");
            }
        }

        if let Some(expect_no_changes) = options.expect_no_changes {
            if expect_no_changes {
                args.push("--expect-no-changes");
            }
        }

        if let Some(show_reads) = options.show_reads {
            if show_reads {
                args.push("--show-reads");
            }
        }

        if let Some(show_replacement_steps) = options.show_replacement_steps {
            if show_replacement_steps {
                args.push("--show-replacement-steps");
            }
        }

        self.workspace
            .run_engine_operation(args, listener.event_tx, listener.preview_tx)
            .await
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        Ok(())
    }

    #[instrument(skip(self))]
    async fn refresh(
        &self,
        options: crate::stack::StackRefreshOptions,
        listener: crate::stack::PulumiProcessListener,
    ) -> crate::Result<()> {
        let mut args = vec![
            "refresh",
            "--stack",
            &self.name,
            "--non-interactive",
            "--json",
            "--diff",
        ];

        if let Some(skip_preview) = options.skip_preview {
            if skip_preview {
                args.push("--skip-preview");
            }
        } else {
            args.push("--yes");
        }

        if let Some(target) = options.target.as_ref() {
            for t in target {
                args.push("--target");
                args.push(&t);
            }
        }

        if let Some(target_dependents) = options.target_dependents {
            if target_dependents {
                args.push("--target-dependents");
            }
        }

        if let Some(exclude) = options.exclude.as_ref() {
            for e in exclude {
                args.push("--exclude");
                args.push(&e);
            }
        }

        if let Some(exclude_dependents) = options.exclude_dependents {
            if exclude_dependents {
                args.push("--exclude-dependents");
            }
        }

        if let Some(show_secrets) = options.show_secrets {
            if show_secrets {
                args.push("--show-secrets");
            }
        }

        if let Some(continue_on_error) = options.continue_on_error {
            if continue_on_error {
                args.push("--continue-on-error");
            }
        }

        self.workspace
            .run_engine_operation(args, listener.event_tx, listener.preview_tx)
            .await
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        Ok(())
    }

    #[instrument(skip(self))]
    async fn destroy(
        &self,
        options: crate::stack::StackDestroyOptions,
        listener: crate::stack::PulumiProcessListener,
    ) -> crate::Result<()> {
        let mut args = vec![
            "destroy",
            "--stack",
            &self.name,
            "--non-interactive",
            "--json",
            "--diff",
        ];

        if let Some(skip_preview) = options.skip_preview {
            if skip_preview {
                args.push("--skip-preview");
            }
        } else {
            args.push("--yes");
        }

        if let Some(refresh) = options.refresh {
            if refresh {
                args.push("--refresh");
            }
        }

        if let Some(target) = options.target.as_ref() {
            for t in target {
                args.push("--target");
                args.push(&t);
            }
        }

        if let Some(target_dependents) = options.target_dependents {
            if target_dependents {
                args.push("--target-dependents");
            }
        }

        if let Some(exclude) = options.exclude.as_ref() {
            for e in exclude {
                args.push("--exclude");
                args.push(&e);
            }
        }

        if let Some(exclude_dependents) = options.exclude_dependents {
            if exclude_dependents {
                args.push("--exclude-dependents");
            }
        }

        if let Some(show_secrets) = options.show_secrets {
            if show_secrets {
                args.push("--show-secrets");
            }
        }

        if let Some(continue_on_error) = options.continue_on_error {
            if continue_on_error {
                args.push("--continue-on-error");
            }
        }

        if let Some(exclude_protected) = options.exclude_protected {
            if exclude_protected {
                args.push("--exclude-protected");
            }
        }

        self.workspace
            .run_engine_operation(args, listener.event_tx, listener.preview_tx)
            .await
            .map_err(|e| {
                super::PulumiError::Other(format!("Failed to run pulumi command: {:?}", e))
            })?;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    #[test]
    fn unit_can_run() {
        assert!(
            true,
            "This test is a placeholder and should be replaced with actual tests."
        );
    }
}
