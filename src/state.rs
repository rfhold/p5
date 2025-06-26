use std::collections::HashMap;

use pulumi_automation::{
    event::{EngineEvent, EventType, ResOpFailedDetails, ResOutputsDetails, ResourcePreDetails},
    local::{LocalStack, LocalWorkspace},
    stack::StackChangeSummary,
    workspace::{Deployment, OutputMap, StackSettings, StackSummary},
};
use ratatui::widgets::ListState;
use tui_input::Input;

use crate::widgets::ResourceListState;

type Result<T> = std::result::Result<T, String>;

#[derive(Default)]
pub struct AppState {
    pub context_stack: Vec<AppContext>,
    pub command_prompt: Input,
    pub selected_workspace: Option<WorkspaceState>,

    pub workspace_list_state: ListState,
    pub stack_list_state: ListState,

    pub toast: Option<(chrono::DateTime<chrono::Utc>, String)>,

    pub workspaces: Loadable<Vec<LocalWorkspace>>,

    /// workspace paths to their outputs
    pub workspace_store: HashMap<String, WorkspaceOutputs>,
}

impl AppState {
    pub fn push_context(&mut self, context: AppContext) {
        if let Some(current_context) = self.context_stack.last() {
            if current_context == &context {
                return; // No need to push the same context again
            }
        }

        match context {
            AppContext::WorkspaceList => {
                self.context_stack.clear();
            }
            AppContext::StackList => {
                self.context_stack.clear();
                self.context_stack.push(AppContext::WorkspaceList);
            }
            AppContext::Stack(ref stack_context) => {
                // Check if we're pushing Details context and push instead of clear
                match stack_context {
                    StackContext::Operation(OperationContext::Summary(
                        OperationDetailsContent::Details,
                    ))
                    | StackContext::Operation(OperationContext::Events(
                        OperationDetailsContent::Details,
                    )) => {
                        // For Details context, just push without clearing to preserve navigation stack
                        // This allows users to go back from details view
                    }
                    _ => {
                        // Default behavior for non-operation stack contexts
                        self.context_stack.clear();
                        self.context_stack.push(AppContext::WorkspaceList);
                        self.context_stack.push(AppContext::StackList);
                    }
                }
            }
            _ => {}
        }

        self.context_stack.push(context);
    }

    pub fn try_selected_workspace(&self) -> Option<LocalWorkspace> {
        if let Some(i) = self.workspace_list_state.selected() {
            if let Some(workspace) = self.workspaces.as_option().and_then(|ws| ws.get(i)) {
                return Some(workspace.clone());
            }
        }

        None
    }

    pub fn try_selected_stack(&self) -> Option<StackSummary> {
        if let Some(state) = &self.selected_workspace {
            if let Some(i) = self.stack_list_state.selected() {
                if let Some(outputs) = self.workspace_store.get(&state.workspace_path) {
                    if let Some(stack) = outputs.stacks.as_option().and_then(|stacks| stacks.get(i))
                    {
                        return Some(stack.clone());
                    }
                }
            }
        }

        None
    }

    pub fn select_workspace_by_cwd(&mut self, cwd: &str) {
        if let Some(i) = self
            .workspaces
            .as_option()
            .and_then(|ws| ws.iter().position(|w| w.cwd == cwd))
        {
            self.workspace_list_state.select(Some(i));
        }

        self.selected_workspace = Some(WorkspaceState {
            workspace_path: cwd.to_string(),
            selected_stack: None,
        });
    }

    pub fn select_stack_by_name_and_cwd(&mut self, stack_name: &str, cwd: &str) {
        self.select_workspace_by_cwd(cwd);

        if let Some(state) = self.selected_workspace.as_mut() {
            if let Some(outputs) = self.workspace_store.get_mut(&state.workspace_path) {
                if let Some(i) = outputs
                    .stacks
                    .as_option()
                    .and_then(|stacks| stacks.iter().position(|s| s.name == stack_name))
                {
                    self.stack_list_state.select(Some(i));
                }
            }

            state.selected_stack = Some(StackState {
                stack_name: stack_name.to_string(),
                resource_state: ResourceListState::default(),
            });
        }
    }

    pub fn background_context(&self) -> AppContext {
        if let Some(context) = self.context_stack.last() {
            if let AppContext::CommandPrompt = context {
                if self.context_stack.len() > 1 {
                    return self.context_stack[self.context_stack.len() - 2].clone();
                }
                return Default::default();
            }
            return context.clone();
        }
        Default::default()
    }

    pub fn current_context(&self) -> AppContext {
        self.context_stack.last().cloned().unwrap_or_default()
    }

    pub fn workspace(&self) -> &Loadable<LocalWorkspace> {
        if let Some(state) = &self.selected_workspace {
            if let Some(outputs) = self.workspace_store.get(&state.workspace_path) {
                return &outputs.workspace;
            }
        }

        &Loadable::NotLoaded
    }

    pub fn stack_state_mut(&mut self) -> Option<&mut StackOutputs> {
        if let Some(state) = self.selected_workspace.as_mut() {
            if let Some(outputs) = self.workspace_store.get_mut(&state.workspace_path) {
                if let Some(stack_name) = state.selected_stack.as_mut() {
                    if let Some(stack_outputs) = outputs.stack_store.get_mut(&stack_name.stack_name)
                    {
                        return Some(stack_outputs);
                    }
                }
            }
        }

        None
    }

    pub fn stack_state(&self) -> Option<&StackOutputs> {
        if let Some(state) = self.selected_workspace.as_ref() {
            if let Some(outputs) = self.workspace_store.get(&state.workspace_path) {
                if let Some(stack_name) = state.selected_stack.as_ref() {
                    if let Some(stack_outputs) = outputs.stack_store.get(&stack_name.stack_name) {
                        return Some(stack_outputs);
                    }
                }
            }
        }

        None
    }

    pub fn stack(&self) -> &Loadable<LocalStack> {
        if let Some(stack_outputs) = self.stack_state() {
            return &stack_outputs.stack;
        }

        &Loadable::NotLoaded
    }

    pub fn stack_outputs(&self) -> &Loadable<OutputMap> {
        if let Some(stack_outputs) = self.stack_state() {
            return &stack_outputs.outputs;
        }

        &Loadable::NotLoaded
    }

    pub fn stack_config(&self) -> &Loadable<StackSettings> {
        if let Some(stack_outputs) = self.stack_state() {
            return &stack_outputs.config;
        }

        &Loadable::NotLoaded
    }

    pub fn stack_state_data(&self) -> &Loadable<Deployment> {
        if let Some(stack_outputs) = self.stack_state() {
            return &stack_outputs.state;
        }

        &Loadable::NotLoaded
    }

    pub fn stack_resource_state(
        &mut self,
    ) -> Option<(&Loadable<Deployment>, &mut ResourceListState)> {
        if let Some(workspace_state) = self.selected_workspace.as_mut() {
            if let Some(stack_name) = workspace_state.selected_stack.as_mut() {
                if let Some(stack_outputs) = self
                    .workspace_store
                    .get_mut(&workspace_state.workspace_path)
                {
                    if let Some(stack_outputs) =
                        stack_outputs.stack_store.get_mut(&stack_name.stack_name)
                    {
                        return Some((&stack_outputs.state, &mut stack_name.resource_state));
                    }
                }
            }
        }
        None
    }

    pub fn stack_operation_state(
        &mut self,
    ) -> Option<(&mut OperationProgress, &mut ResourceListState)> {
        if let Some(workspace_state) = self.selected_workspace.as_mut() {
            if let Some(stack_name) = workspace_state.selected_stack.as_mut() {
                if let Some(stack_outputs) = self
                    .workspace_store
                    .get_mut(&workspace_state.workspace_path)
                {
                    if let Some(stack_outputs) =
                        stack_outputs.stack_store.get_mut(&stack_name.stack_name)
                    {
                        if let Some(operation_progress) = &mut stack_outputs.operation {
                            return Some((operation_progress, &mut stack_name.resource_state));
                        }
                    }
                }
            }
        }
        None
    }

    pub fn operation_progress(&self) -> Option<&OperationProgress> {
        if let Some(stack_outputs) = self.stack_state() {
            return stack_outputs.operation.as_ref();
        }
        None
    }

    pub fn stacks(&self) -> &Loadable<Vec<StackSummary>> {
        if let Some(state) = &self.selected_workspace {
            if let Some(outputs) = self.workspace_store.get(&state.workspace_path) {
                return &outputs.stacks;
            }
        }

        &Loadable::NotLoaded
    }

    pub fn workspaces(&self) -> &Loadable<Vec<LocalWorkspace>> {
        &self.workspaces
    }

    pub fn stack_context(&self) -> StackContext {
        if let AppContext::Stack(stack_context) = self.background_context() {
            return stack_context.clone();
        }
        StackContext::Config
    }
}

#[derive(Debug, Clone, Default)]
pub enum Loadable<T> {
    #[default]
    NotLoaded,
    Loading,
    Loaded(T),
}

impl<T> Loadable<T> {
    pub fn is_loaded(&self) -> bool {
        matches!(self, Loadable::Loaded(_))
    }

    pub fn is_loading(&self) -> bool {
        matches!(self, Loadable::Loading)
    }

    pub fn is_not_loaded(&self) -> bool {
        matches!(self, Loadable::NotLoaded)
    }

    pub fn as_mut_or_default(&mut self, default: T) -> &mut T {
        match self {
            Loadable::Loaded(value) => value,
            Loadable::Loading | Loadable::NotLoaded => {
                *self = Loadable::Loaded(default);
                if let Loadable::Loaded(value) = self {
                    value
                } else {
                    unreachable!()
                }
            }
        }
    }

    pub fn as_ref(&self) -> Loadable<&T> {
        match self {
            Loadable::Loaded(value) => Loadable::Loaded(value),
            Loadable::Loading => Loadable::Loading,
            Loadable::NotLoaded => Loadable::NotLoaded,
        }
    }

    pub fn as_option(&self) -> Option<&T> {
        match self {
            Loadable::Loaded(value) => Some(value),
            Loadable::Loading | Loadable::NotLoaded => None,
        }
    }
}

#[derive(Default)]
pub struct WorkspaceOutputs {
    pub workspace: Loadable<LocalWorkspace>,
    pub stacks: Loadable<Vec<StackSummary>>,
    /// stack names to their outputs
    pub stack_store: HashMap<String, StackOutputs>,
}

#[derive(Default)]
pub struct StackOutputs {
    pub stack: Loadable<LocalStack>,
    pub outputs: Loadable<OutputMap>,
    pub config: Loadable<StackSettings>,
    pub state: Loadable<Deployment>,
    pub operation: Option<OperationProgress>,
}

#[derive(Clone, Debug)]
pub enum ProgramOperation {
    Update,
    Destroy,
    Refresh,
}

#[derive(Default, Debug)]
pub struct WorkspaceState {
    pub workspace_path: String,
    pub selected_stack: Option<StackState>,
}

#[derive(Debug)]
pub struct StackState {
    pub stack_name: String,
    pub resource_state: ResourceListState,
}

#[derive(Clone, Default, Debug, Eq, PartialEq)]
pub enum AppContext {
    CommandPrompt,
    #[default]
    WorkspaceList,
    StackList,
    Stack(StackContext),
}

#[derive(Clone, Default, Debug, Eq, PartialEq)]
pub enum StackContext {
    Outputs,
    #[default]
    Config,
    Resources,
    Operation(OperationContext),
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub enum OperationContext {
    Summary(OperationDetailsContent),
    Events(OperationDetailsContent),
}

#[derive(Clone, Copy, Debug, Eq, PartialEq, Default)]
pub enum OperationDetailsContent {
    #[default]
    List,
    Details,
    Item,
}

#[derive(Debug, Clone)]
pub struct OperationProgress {
    pub operation: ProgramOperation,
    pub options: Option<OperationOptions>,
    // loaded before executing for user review
    pub change_summary: Loadable<StackChangeSummary>,
    // loaded during execution
    pub events: Loadable<OperationEvents>,
}

impl OperationProgress {
    pub fn is_preview(&self) -> bool {
        if let Some(options) = &self.options {
            options.preview_only
        } else {
            false
        }
    }

    pub fn is_skip_preview(&self) -> bool {
        if let Some(options) = &self.options {
            options.skip_preview
        } else {
            false
        }
    }
}

#[derive(Clone, Default, Debug)]
pub struct OperationOptions {
    pub show_replacement_steps: bool,
    pub preview_only: bool,
    pub skip_preview: bool,
}

#[derive(Debug, Clone, Default)]
pub struct OperationEvents {
    pub events: Vec<EngineEvent>,
    pub states: Vec<ResourceOperationState>,
    pub done: bool,
}

#[derive(Debug, Clone)]
pub enum ResourceOperationState {
    InProgress {
        sequence: i64,
        start_time: chrono::DateTime<chrono::Utc>,
        pre_event: ResourcePreDetails,
    },
    Completed {
        sequence: i64,
        start_time: chrono::DateTime<chrono::Utc>,
        end_time: chrono::DateTime<chrono::Utc>,
        pre_event: ResourcePreDetails,
        out_event: ResOutputsDetails,
    },
    Failed {
        sequence: i64,
        start_time: chrono::DateTime<chrono::Utc>,
        end_time: chrono::DateTime<chrono::Utc>,
        pre_event: ResourcePreDetails,
        failed_event: ResOpFailedDetails,
    },
}

impl OperationEvents {
    fn find_in_progress_state_mut(&mut self, urn: &str) -> Result<&mut ResourceOperationState> {
        if let Some((index, _)) = self.states.iter().enumerate().find(|(_, state)| {
            if let ResourceOperationState::InProgress { pre_event, .. } = state {
                pre_event.metadata.urn == urn
            } else {
                false
            }
        }) {
            Ok(&mut self.states[index])
        } else {
            Err("InProgress state not found for the given URN".to_string())
        }
    }

    pub fn apply_event(
        &mut self,
        event: EngineEvent,
        options: Option<OperationOptions>,
    ) -> Result<()> {
        self.events.push(event.clone());

        if let Some(opts) = options {
            if !opts.show_replacement_steps && event.event.is_replacement_step_event() {
                // Skip replacement steps if the option is set
                return Ok(());
            }
        }

        let event_time = event
            .timestamp
            .map_or(Some(chrono::Utc::now()), |t| {
                chrono::DateTime::from_timestamp(t, 0)
            })
            .unwrap_or_default();

        match event.event {
            EventType::ResourcePreEvent { details, .. } => {
                let state = ResourceOperationState::InProgress {
                    sequence: event.sequence.unwrap_or_default(),
                    start_time: event_time,
                    pre_event: details,
                };
                self.states.push(state);
            }
            EventType::ResOutputsEvent { details, .. } => {
                let urn = &details.metadata.urn;
                let state = self.find_in_progress_state_mut(urn)?;

                // Transform the InProgress state into a Completed state
                if let ResourceOperationState::InProgress {
                    sequence,
                    start_time,
                    pre_event,
                } = state.clone()
                {
                    *state = ResourceOperationState::Completed {
                        sequence,
                        start_time,
                        end_time: event_time,
                        pre_event,
                        out_event: details,
                    };
                }
            }
            EventType::ResOpFailedEvent { details, .. } => {
                let urn = &details.metadata.urn;
                let state = self.find_in_progress_state_mut(urn)?;

                // Transform the InProgress state into a Failed state
                if let ResourceOperationState::InProgress {
                    sequence,
                    start_time,
                    pre_event,
                } = state.clone()
                {
                    *state = ResourceOperationState::Failed {
                        sequence,
                        start_time,
                        end_time: event_time,
                        pre_event,
                        failed_event: details,
                    };
                }
            }
            _ => {}
        }

        Ok(())
    }
}

#[cfg(test)]
pub mod tests {
    use pulumi_automation::{
        event::EngineEvent,
        local::{LocalStack, LocalWorkspace},
        stack::StackChangeSummary,
    };
    use std::collections::HashMap;

    use crate::widgets::ResourceListState;

    fn relative_file(path: &str) -> std::path::PathBuf {
        let manifest_dir = std::env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR not set");

        std::path::PathBuf::from(manifest_dir)
            .join("src/test/fixtures")
            .join(path)
    }

    fn load_preview_from_fixture(fixture: &str) -> Result<StackChangeSummary, String> {
        let path = relative_file(fixture);
        let content = std::fs::read_to_string(path).map_err(|e| e.to_string())?;
        serde_json::from_str(&content).map_err(|e| e.to_string())
    }

    fn load_events_from_fixture(fixture: &str) -> Result<Vec<EngineEvent>, String> {
        let path = relative_file(fixture);
        let content = std::fs::read_to_string(path).map_err(|e| e.to_string())?;

        content
            .lines()
            .map(|line| serde_json::from_str(line).map_err(|e| e.to_string()))
            .collect()
    }

    /// Create a complete AppState loaded with fixture data for layout testing
    pub fn create_test_app_state_with_fixtures(complete: bool) -> super::AppState {
        let mut state = super::AppState::default();

        // Load test workspaces
        let workspaces = vec![LocalWorkspace {
            cwd: "test-workspace".to_string(),
        }];
        state.workspaces = super::Loadable::Loaded(workspaces);

        // Create a test workspace state
        state.selected_workspace = Some(super::WorkspaceState {
            workspace_path: "test-workspace".to_string(),
            selected_stack: Some(super::StackState {
                stack_name: "test-stack".to_string(),
                resource_state: ResourceListState::default(),
            }),
        });

        // Load fixture data into workspace store
        let mut workspace_outputs = super::WorkspaceOutputs {
            workspace: super::Loadable::NotLoaded,
            stacks: super::Loadable::Loaded(vec![super::StackSummary {
                name: "test-stack".to_string(),
                last_update: None,
                ..Default::default()
            }]),
            stack_store: HashMap::new(),
        };

        // Try to load preview and events from fixtures
        let preview = load_preview_from_fixture("preview.json").unwrap();

        let stack_outputs = super::StackOutputs {
            stack: super::Loadable::Loaded(LocalStack {
                name: "test-stack".to_string(),
                workspace: LocalWorkspace {
                    cwd: "test-workspace".to_string(),
                },
            }),
            outputs: super::Loadable::NotLoaded,
            config: super::Loadable::NotLoaded,
            state: super::Loadable::NotLoaded,
            operation: None,
        };
        workspace_outputs
            .stack_store
            .insert("test-stack".to_string(), stack_outputs);

        let events = load_events_from_fixture("success-events.json").unwrap();
        let mut operation_events = super::OperationEvents {
            events: events.clone(),
            states: Vec::new(),
            done: complete,
        };

        let options = super::OperationOptions {
            show_replacement_steps: false,
            preview_only: false,
            skip_preview: false,
        };

        // Apply events to build operation states
        for event in events {
            operation_events
                .apply_event(event, Some(options.clone()))
                .unwrap();
        }

        let operation_progress = super::OperationProgress {
            operation: super::ProgramOperation::Update,
            options: Some(options),
            change_summary: super::Loadable::Loaded(preview),
            events: super::Loadable::Loaded(operation_events),
        };

        if let Some(stack_outputs) = workspace_outputs.stack_store.get_mut("test-stack") {
            stack_outputs.operation = Some(operation_progress);
        }

        state
            .workspace_store
            .insert("test-workspace".to_string(), workspace_outputs);

        // Set up default context stack
        state
            .context_stack
            .push(super::AppContext::Stack(super::StackContext::Operation(
                super::OperationContext::Summary(super::OperationDetailsContent::List),
            )));

        state
    }

    /// Helper struct to analyze operation events for testing
    #[derive(Debug)]
    struct OperationAnalysis {
        /// Total number of states
        pub total_states: usize,
        /// States grouped by URN
        pub states_by_urn: HashMap<String, Vec<super::ResourceOperationState>>,
        /// States grouped by operation type
        pub states_by_operation: HashMap<String, Vec<super::ResourceOperationState>>,
        /// Count of replacement operations (create-replacement, delete-replaced, replace)
        pub replacement_operation_count: usize,
    }

    impl OperationAnalysis {
        fn from_operation_events(operation_events: &super::OperationEvents) -> Self {
            let mut states_by_urn = HashMap::new();
            let mut states_by_operation = HashMap::new();
            let mut replacement_operation_count = 0;

            for state in &operation_events.states {
                let (urn, op) = match state {
                    super::ResourceOperationState::InProgress { pre_event, .. } => (
                        pre_event.metadata.urn.clone(),
                        pre_event.metadata.op.to_string(),
                    ),
                    super::ResourceOperationState::Completed { pre_event, .. } => (
                        pre_event.metadata.urn.clone(),
                        pre_event.metadata.op.to_string(),
                    ),
                    super::ResourceOperationState::Failed { pre_event, .. } => (
                        pre_event.metadata.urn.clone(),
                        pre_event.metadata.op.to_string(),
                    ),
                };

                // Count replacement operations
                if op.to_lowercase().contains("replacement")
                    || op.to_lowercase().contains("replace")
                    || op.to_lowercase().contains("deleted")
                {
                    replacement_operation_count += 1;
                }

                // Group by URN
                states_by_urn
                    .entry(urn.clone())
                    .or_insert_with(Vec::new)
                    .push(state.clone());

                // Group by operation type
                states_by_operation
                    .entry(op)
                    .or_insert_with(Vec::new)
                    .push(state.clone());
            }

            Self {
                total_states: operation_events.states.len(),
                states_by_urn,
                states_by_operation,
                replacement_operation_count,
            }
        }

        /// Get the number of unique resources (URNs) involved
        pub fn unique_resource_count(&self) -> usize {
            self.states_by_urn.len()
        }

        /// Get resources that have multiple states (potential candidates for combining)
        pub fn resources_with_multiple_states(&self) -> HashMap<String, usize> {
            self.states_by_urn
                .iter()
                .filter(|(_, states)| states.len() > 1)
                .map(|(urn, states)| (urn.clone(), states.len()))
                .collect()
        }

        /// Print detailed analysis for debugging
        pub fn print_analysis(&self) {
            println!("=== Operation Events Analysis ===");
            println!("Total states: {}", self.total_states);
            println!("Unique resources: {}", self.unique_resource_count());
            println!(
                "Replacement operations: {}",
                self.replacement_operation_count
            );

            println!("\n--- States by Operation Type ---");
            for (op, states) in &self.states_by_operation {
                println!("{}: {} states", op, states.len());
            }

            println!("\n--- Resources with Multiple States ---");
            for (urn, count) in self.resources_with_multiple_states() {
                println!("{}: {} states", urn, count);
            }

            println!("\n--- Detailed State Breakdown ---");
            for (urn, states) in &self.states_by_urn {
                if states.len() > 1 {
                    println!("\nResource: {}", urn);
                    for (i, state) in states.iter().enumerate() {
                        let (sequence, op) = match state {
                            super::ResourceOperationState::InProgress {
                                sequence,
                                pre_event,
                                ..
                            } => (*sequence, pre_event.metadata.op.to_string()),
                            super::ResourceOperationState::Completed {
                                sequence,
                                pre_event,
                                ..
                            } => (*sequence, pre_event.metadata.op.to_string()),
                            super::ResourceOperationState::Failed {
                                sequence,
                                pre_event,
                                ..
                            } => (*sequence, pre_event.metadata.op.to_string()),
                        };
                        println!("  State {}: seq={}, op={}", i + 1, sequence, op);
                    }
                }
            }
        }
    }

    /// Test framework for asserting expected state counts
    struct StateCountAssertions {
        /// Expected total number of states after optimization
        pub expected_total_states: Option<usize>,
        /// Expected number of states per resource (URN -> expected count)
        pub expected_states_per_resource: HashMap<String, usize>,
        /// Expected number of states per operation type
        pub expected_states_per_operation: HashMap<String, usize>,
    }

    impl StateCountAssertions {
        fn new() -> Self {
            Self {
                expected_total_states: None,
                expected_states_per_resource: HashMap::new(),
                expected_states_per_operation: HashMap::new(),
            }
        }

        fn expect_total_states(mut self, count: usize) -> Self {
            self.expected_total_states = Some(count);
            self
        }

        fn expect_states_for_resource(mut self, urn: &str, count: usize) -> Self {
            self.expected_states_per_resource
                .insert(urn.to_string(), count);
            self
        }

        fn expect_states_for_operation(mut self, operation: &str, count: usize) -> Self {
            self.expected_states_per_operation
                .insert(operation.to_string(), count);
            self
        }

        fn assert_against(&self, analysis: &OperationAnalysis) {
            if let Some(expected_total) = self.expected_total_states {
                assert_eq!(
                    analysis.total_states, expected_total,
                    "Expected {} total states, but got {}",
                    expected_total, analysis.total_states
                );
            }

            for (urn, expected_count) in &self.expected_states_per_resource {
                let actual_count = analysis
                    .states_by_urn
                    .get(urn)
                    .map(|states| states.len())
                    .unwrap_or(0);
                assert_eq!(
                    actual_count, *expected_count,
                    "Expected {} states for resource '{}', but got {}",
                    expected_count, urn, actual_count
                );
            }

            for (operation, expected_count) in &self.expected_states_per_operation {
                let actual_count = analysis
                    .states_by_operation
                    .get(operation)
                    .map(|states| states.len())
                    .unwrap_or(0);
                assert_eq!(
                    actual_count, *expected_count,
                    "Expected {} states for operation '{}', but got {}",
                    expected_count, operation, actual_count
                );
            }
        }
    }

    #[test]
    fn test_operation_events_success_analysis() {
        let events = load_events_from_fixture("success-events.json").unwrap();
        let mut operation_events = super::OperationEvents::default();

        for event in events {
            operation_events.apply_event(event, None).unwrap();
        }

        let analysis = OperationAnalysis::from_operation_events(&operation_events);
        analysis.print_analysis();

        let baseline_assertions = StateCountAssertions::new().expect_total_states(20); // This is what we currently get

        baseline_assertions.assert_against(&analysis);

        // Enable detailed assertions to validate the filtering behavior
        let detailed_assertions = StateCountAssertions::new()
            .expect_total_states(20) // Current state after filtering
            .expect_states_for_resource("urn:pulumi:base-reference::debug::local:index/file:File::iteration", 3)
            .expect_states_for_resource("urn:pulumi:base-reference::debug::local:index/file:File::secret", 3)
            .expect_states_for_resource("urn:pulumi:base-reference::debug::random:index/randomPassword:RandomPassword::password", 3)
            .expect_states_for_resource("urn:pulumi:base-reference::debug::local:index/file:File::file-from-command", 3)
            .expect_states_for_operation("CreateReplacement", 4)
            .expect_states_for_operation("DeleteReplaced", 4)
            .expect_states_for_operation("Replace", 4);

        detailed_assertions.assert_against(&analysis);
    }

    #[test]
    fn test_operation_events_replacement_combining() {
        let events = load_events_from_fixture("success-events.json").unwrap();
        let mut operation_events = super::OperationEvents::default();

        for event in events {
            operation_events.apply_event(event, None).unwrap();
        }

        let analysis = OperationAnalysis::from_operation_events(&operation_events);

        // Log the current state for debugging
        println!("Before optimization:");
        println!("- Total states: {}", analysis.total_states);
        println!(
            "- Replacement operations: {}",
            analysis.replacement_operation_count
        );
        println!(
            "- Resources with multiple states: {:?}",
            analysis.resources_with_multiple_states()
        );

        let replacement_assertions = StateCountAssertions::new()
            .expect_total_states(20)
            .expect_states_for_operation("CreateReplacement", 4)
            .expect_states_for_operation("DeleteReplaced", 4)
            .expect_states_for_operation("Replace", 4)
            .expect_states_for_operation("Delete", 2)
            .expect_states_for_operation("Create", 1)
            .expect_states_for_operation("Update", 1);

        replacement_assertions.assert_against(&analysis);

        assert_eq!(analysis.total_states, 20);
        assert_eq!(
            analysis.replacement_operation_count, 12,
            "Should have exactly 12 replacement operations"
        );
        assert_eq!(
            analysis.unique_resource_count(),
            12,
            "Should have 12 unique resources"
        );

        // Validate specific resources with multiple states
        let multiple_states = analysis.resources_with_multiple_states();
        assert_eq!(
            multiple_states.len(),
            4,
            "Should have 4 resources with multiple states"
        );
    }

    #[test]
    fn test_operation_events_failed() {
        let events = load_events_from_fixture("failed-events.json").unwrap();
        let mut operation_events = super::OperationEvents::default();

        for event in events {
            operation_events.apply_event(event, None).unwrap();
        }

        let analysis = OperationAnalysis::from_operation_events(&operation_events);
        analysis.print_analysis();

        assert!(!operation_events.events.is_empty());
        assert!(!operation_events.states.is_empty());
    }

    #[test]
    fn test_operation_events_hide_replacement_steps() {
        let events = load_events_from_fixture("success-events.json").unwrap();
        let mut operation_events = super::OperationEvents::default();

        let options = super::OperationOptions {
            show_replacement_steps: false,
            preview_only: false,
            skip_preview: false,
        };

        for event in events {
            operation_events
                .apply_event(event, Some(options.clone()))
                .unwrap();
        }

        // Analyze the operation events after filtering replacements
        let analysis = OperationAnalysis::from_operation_events(&operation_events);
        analysis.print_analysis();

        // Validate that replacement steps are filtered out
        assert!(
            analysis.replacement_operation_count < 12,
            "Replacement operations should be filtered out"
        );
    }
}
