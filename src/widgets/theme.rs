pub mod color {
    use ratatui::style::Color;

    pub const BORDER_DEFAULT: Color = Color::Gray;
    pub const BORDER_HIGHLIGHT: Color = Color::LightBlue;
    pub const TEXT_DEFAULT: Color = Color::Gray;

    pub const SELECTED: Color = Color::LightMagenta;

    pub const ATTENTION_NIL: Color = Color::DarkGray;
    pub const ATTENTION_DESTROY: Color = Color::Red;
    pub const ATTENTION_DISCARD: Color = Color::LightRed;
    pub const ATTENTION_WRITE: Color = Color::Yellow;
    pub const ATTENTION_READ: Color = Color::Blue;
    pub const ATTENTION_CREATE: Color = Color::Green;
    pub const ATTENTION_IMPORT: Color = Color::LightGreen;
    pub const ATTENTION_REPLACE: Color = Color::Magenta;
}
