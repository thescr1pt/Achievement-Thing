pub mod parser {
    use ini::Ini;
    pub fn parse_file(filename: &str) -> Result<Vec<String>, String> {
        match filename.split('.').last() {
            Some("ini") => parse_ini(filename),
            Some("json") => Err("JSON parsing not implemented yet".to_string()),
            _ => Err("Unsupported file format".to_string()),
        }
    }

    pub fn parse_ini(filename: &str) -> Result<Vec<String>, String> {
        let mut achievements: Vec<String> = Vec::new();
        // Load the INI file
        let conf = match Ini::load_from_file(filename).map_err(|e| e.to_string()) {
            Ok(conf) => conf,
            Err(e) => return Err(format!("Failed to load INI file: {}", e)),
        };
        for (sec, _) in &conf {
            //check if section name is not empy or is not called "steamachievements"
            if !sec.is_empty() && sec != "steamachievements" {
                achievements.push(sec.clone());
            }
        }
        
        return Ok(achievements);
    }

}