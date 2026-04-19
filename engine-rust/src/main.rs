use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
struct CharlieEvent {
    event_id: String,
    event_type: String,
    charge_point_id: String,
    payload: serde_json::Value,
    timestamp: String,
}

fn main() {
    println!("🦀 Charlie Rust Engine starting up...");
    // Future: Connect to Kafka or HTTP Server here
}