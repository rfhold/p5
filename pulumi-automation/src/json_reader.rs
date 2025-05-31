use serde::de::DeserializeOwned;
use tokio::io::{AsyncBufReadExt, Lines};
use tokio::sync::mpsc::{self, Receiver};

/// Parses a stream of multi-line JSON objects from a line reader
///
/// This function handles JSON objects that span multiple lines by tracking
/// brace counts and string literals to determine when a complete JSON object
/// has been received.
pub fn parse_json_stream<R, T>(mut lines: Lines<R>) -> Receiver<Result<T, JsonStreamError>>
where
    R: AsyncBufReadExt + Unpin + Send + 'static,
    T: DeserializeOwned + Send + 'static,
{
    let (tx, rx) = mpsc::channel(100);

    tokio::spawn(async move {
        let mut json_buffer = String::new();
        let mut brace_count = 0;
        let mut in_string = false;
        let mut escape_next = false;

        while let Ok(Some(line)) = lines.next_line().await {
            // Add the line to our buffer
            if !json_buffer.is_empty() {
                json_buffer.push('\n');
            }
            json_buffer.push_str(&line);

            // Parse the line character by character to track JSON structure
            for ch in line.chars() {
                if escape_next {
                    escape_next = false;
                    continue;
                }

                match ch {
                    '\\' if in_string => escape_next = true,
                    '"' if !escape_next => in_string = !in_string,
                    '{' if !in_string => brace_count += 1,
                    '}' if !in_string => {
                        brace_count -= 1;

                        // Check if we have a complete JSON object
                        if brace_count == 0 && !json_buffer.trim().is_empty() {
                            let result = serde_json::from_str::<T>(&json_buffer).map_err(|e| {
                                JsonStreamError::ParseError {
                                    error: e.to_string(),
                                    json: json_buffer.clone(),
                                }
                            });

                            // Send the result regardless of success/failure
                            if tx.send(result).await.is_err() {
                                // Receiver dropped, exit
                                break;
                            }

                            json_buffer.clear();
                        }
                    }
                    _ => {}
                }
            }

            // Handle malformed JSON - if brace count goes negative
            if brace_count < 0 {
                let _ = tx
                    .send(Err(JsonStreamError::MalformedJson {
                        message: "Unmatched closing brace".to_string(),
                        buffer: json_buffer.clone(),
                    }))
                    .await;

                // Reset state
                json_buffer.clear();
                brace_count = 0;
                in_string = false;
                escape_next = false;
            }
        }

        // Handle any remaining content in the buffer
        if !json_buffer.trim().is_empty() {
            let _ = tx
                .send(Err(JsonStreamError::IncompleteJson {
                    buffer: json_buffer,
                    brace_count,
                }))
                .await;
        }
    });

    rx
}

/// Errors that can occur during JSON stream parsing
#[derive(Debug, Clone)]
pub enum JsonStreamError {
    ParseError { error: String, json: String },
    MalformedJson { message: String, buffer: String },
    IncompleteJson { buffer: String, brace_count: i32 },
}

impl std::fmt::Display for JsonStreamError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            JsonStreamError::ParseError { error, json } => {
                write!(f, "Failed to parse JSON: {}\nJSON: {}", error, json)
            }
            JsonStreamError::MalformedJson { message, buffer } => {
                write!(f, "Malformed JSON: {}\nBuffer: {}", message, buffer)
            }
            JsonStreamError::IncompleteJson {
                buffer,
                brace_count,
            } => {
                write!(
                    f,
                    "Incomplete JSON (brace count: {})\nBuffer: {}",
                    brace_count, buffer
                )
            }
        }
    }
}

impl std::error::Error for JsonStreamError {}
