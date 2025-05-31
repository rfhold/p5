use serde_json::Value;
use std::collections::{HashMap, HashSet};

#[derive(Debug, Clone, PartialEq, Eq, Hash, PartialOrd, Ord)]
enum JsonType {
    Null,
    Bool,
    Number,
    String,
    Object,
    Array(Box<JsonType>),
    Mixed(Vec<JsonType>),
}

impl JsonType {
    fn from_value(value: &Value) -> Self {
        match value {
            Value::Null => JsonType::Null,
            Value::Bool(_) => JsonType::Bool,
            Value::Number(_) => JsonType::Number,
            Value::String(_) => JsonType::String,
            Value::Object(_) => JsonType::Object,
            Value::Array(arr) => {
                if arr.is_empty() {
                    JsonType::Array(Box::new(JsonType::Null))
                } else {
                    let types: HashSet<JsonType> =
                        arr.iter().map(|v| JsonType::from_value(v)).collect();

                    if types.len() == 1 {
                        JsonType::Array(Box::new(types.into_iter().next().unwrap()))
                    } else {
                        let mut types_vec: Vec<JsonType> = types.into_iter().collect();
                        types_vec.sort();
                        JsonType::Array(Box::new(JsonType::Mixed(types_vec)))
                    }
                }
            }
        }
    }

    fn to_string(&self) -> String {
        match self {
            JsonType::Null => "null".to_string(),
            JsonType::Bool => "bool".to_string(),
            JsonType::Number => "number".to_string(),
            JsonType::String => "string".to_string(),
            JsonType::Object => "object".to_string(),
            JsonType::Array(inner) => format!("array[{}]", inner.to_string()),
            JsonType::Mixed(types) => {
                let types_str: Vec<String> = types.iter().map(|t| t.to_string()).collect();
                types_str.join("|")
            }
        }
    }
}

#[derive(Debug)]
struct PathInfo {
    types: HashSet<JsonType>,
    occurrences: usize,
}

pub fn shake_json_paths(json: Vec<Value>) -> Vec<String> {
    if json.is_empty() {
        return vec![];
    }

    let total_count = json.len();
    let mut path_map: HashMap<String, PathInfo> = HashMap::new();
    let mut array_object_counts: HashMap<String, (usize, usize)> = HashMap::new(); // (object_count, field_count)

    for value in json {
        collect_paths(&value, "", &mut path_map, &mut array_object_counts);
    }

    let mut array_element_paths: HashMap<String, (&PathInfo, bool)> = HashMap::new();
    for (path, info) in &path_map {
        if path.contains("[]") && path != "[]" {
            let base_path = path.split("[]").next().unwrap();
            if let Some(_base_info) = path_map.get(base_path) {
                // Check if this field is optional within the array objects
                let is_optional_in_array =
                    if let Some(&(obj_count, field_count)) = array_object_counts.get(path) {
                        field_count < obj_count
                    } else {
                        false
                    };
                array_element_paths.insert(path.clone(), (info, is_optional_in_array));
            }
        }
    }

    let mut result = Vec::new();

    for (path, info) in &path_map {
        if path.contains("[]") && path != "[]" {
            continue;
        }

        let is_optional = info.occurrences < total_count;
        let type_str = if info.types.len() == 1 {
            info.types.iter().next().unwrap().to_string()
        } else {
            let mut types_vec: Vec<JsonType> = info.types.iter().cloned().collect();
            types_vec.sort();
            JsonType::Mixed(types_vec).to_string()
        };

        let path_str = if path.is_empty() {
            type_str
        } else {
            format!(
                "{}{}{}",
                path,
                if is_optional { "?=" } else { "=" },
                type_str
            )
        };

        result.push(path_str);
    }

    for (path, (info, is_optional_in_array)) in array_element_paths {
        let type_str = if info.types.len() == 1 {
            info.types.iter().next().unwrap().to_string()
        } else {
            let mut types_vec: Vec<JsonType> = info.types.iter().cloned().collect();
            types_vec.sort();
            JsonType::Mixed(types_vec).to_string()
        };

        let path_str = format!(
            "{}{}{}",
            path,
            if is_optional_in_array { "?=" } else { "=" },
            type_str
        );

        result.push(path_str);
    }

    result.sort();

    result
}

fn collect_paths(
    value: &Value,
    current_path: &str,
    path_map: &mut HashMap<String, PathInfo>,
    array_object_counts: &mut HashMap<String, (usize, usize)>,
) {
    let json_type = JsonType::from_value(value);

    let entry = path_map
        .entry(current_path.to_string())
        .or_insert(PathInfo {
            types: HashSet::new(),
            occurrences: 0,
        });
    entry.types.insert(json_type.clone());
    entry.occurrences += 1;

    match value {
        Value::Object(map) => {
            for (key, val) in map {
                let new_path = if current_path.is_empty() {
                    key.clone()
                } else {
                    format!("{}.{}", current_path, key)
                };
                collect_paths(val, &new_path, path_map, array_object_counts);
            }
        }
        Value::Array(arr) => {
            let element_path = format!("{}[]", current_path);
            let mut object_count = 0;

            for element in arr {
                if let Value::Object(obj) = element {
                    object_count += 1;
                    let mut seen_fields = HashSet::new();

                    for (key, val) in obj {
                        let new_path = format!("{}.{}", element_path, key);
                        collect_paths(val, &new_path, path_map, array_object_counts);
                        seen_fields.insert(new_path.clone());
                    }

                    // Update counts for fields seen in this object
                    for field in seen_fields {
                        let entry = array_object_counts.entry(field).or_insert((0, 0));
                        entry.1 += 1;
                    }
                }
            }

            // Update total object count for all fields under this array
            if object_count > 0 {
                let prefix = format!("{}.", element_path);
                for path in path_map.keys() {
                    if path.starts_with(&prefix) {
                        let entry = array_object_counts.entry(path.clone()).or_insert((0, 0));
                        entry.0 = object_count;
                    }
                }
            }
        }
        _ => {}
    }
}

use serde_json::json;

#[test]
fn test_object_single_value() {
    let json_values = vec![json!({"foo": "bar"})];
    let paths = shake_json_paths(json_values);
    assert_eq!(paths, vec!["foo=string", "object"]);
}

#[test]
fn test_object_multiple_values() {
    let json_values = vec![
        json!({"foo": "bar", "baz": 42}),
        json!({"foo": "baz", "biz": true}),
    ];
    let paths = shake_json_paths(json_values);
    assert_eq!(
        paths,
        vec!["baz?=number", "biz?=bool", "foo=string", "object"]
    );
}

#[test]
fn test_object_optional_fields() {
    let json_values = vec![json!({"foo": "bar"}), json!({"foo": "baz", "biz": 42})];
    let paths = shake_json_paths(json_values);
    assert_eq!(
        paths,
        vec![
            "biz?=number", // Optional because not in all objects
            "foo=string",
            "object"
        ]
    );
}

#[test]
fn test_object_nullable_fields() {
    let json_values = vec![
        json!({"foo": "bar", "biz": null}),
        json!({"foo": "baz", "biz": 42}),
    ];
    let paths = shake_json_paths(json_values);
    assert_eq!(
        paths,
        vec![
            "biz=null|number", // Mixed type with null
            "foo=string",
            "object"
        ]
    );
}

#[test]
fn test_array_single_type() {
    let json_values = vec![json!({"items": [1, 2, 3]})];
    let paths = shake_json_paths(json_values);
    assert_eq!(paths, vec!["items=array[number]", "object"]);
}

#[test]
fn test_array_mixed_types() {
    let json_values = vec![
        json!({"items": [1, "two", 3]}),
        json!({"items": [4, 5, "six"]}),
    ];
    let paths = shake_json_paths(json_values);
    assert_eq!(paths, vec!["items=array[number|string]", "object"]);
}

#[test]
fn test_array_object_values() {
    let json_values = vec![json!({"pets": [{"type": "dog"}, {"type": "cat", "name": "mittens"}]})];
    let paths = shake_json_paths(json_values);
    assert_eq!(
        paths,
        vec![
            "object",
            "pets=array[object]",
            "pets[].name?=string", // Optional because not in all objects
            "pets[].type=string"
        ]
    );
}
