CREATE TABLE IF NOT EXISTS `storage` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` TEXT,
  `path` TEXT,
  `text` TEXT,
  `counter` INTEGER NOT NULL DEFAULT 1,
  `hash` VARCHAR(64) NOT NULL,
  `salt_name` VARCHAR(256) NOT NULL,
  `salt_file` VARCHAR(256) NOT NULL,
  `salt_text` VARCHAR(256) NOT NULL,
  `created` DATETIME NOT NULL,
  `updated` DATETIME NOT NULL,
  `expired` DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS `hash` ON `storage` (`hash`);
CREATE INDEX IF NOT EXISTS `expired` ON `storage` (`expired`);