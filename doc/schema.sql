CREATE TABLE IF NOT EXISTS `storage` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `text` TEXT,
  `file_name` TEXT,
  `file_path` TEXT,
  `count_text` INTEGER NOT NULL DEFAULT 1,
  `count_file` INTEGER NOT NULL DEFAULT 1,
  `hash_text` VARCHAR(64) NOT NULL,
  `hash_name` VARCHAR(64) NOT NULL,
  `salt_name` VARCHAR(256) NOT NULL,
  `salt_file` VARCHAR(256) NOT NULL,
  `salt_text` VARCHAR(256) NOT NULL,
  `created` DATETIME NOT NULL,
  `updated` DATETIME NOT NULL,
  `expired` DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS `hash_text` ON `storage` (`hash_text`);
CREATE INDEX IF NOT EXISTS `expired` ON `storage` (`expired`);
