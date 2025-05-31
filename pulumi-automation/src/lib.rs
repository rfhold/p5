pub mod event;
pub mod json_reader;
pub mod local;
pub mod stack;
pub mod workspace;

#[derive(Debug)]
pub enum PulumiError {
    Other(String),
}

pub type Result<T> = std::result::Result<T, PulumiError>;
