pub mod watcher {
    use crate::parser;
    const ACHIEVEMENT_FILES: [&str; 4] = ["achievements.ini","achievements.json","achiev.ini","Achievements.ini"];
    pub fn watch_files() {
        println!("Watching files for changes...");
    }
}

