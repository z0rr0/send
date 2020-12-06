CREATE TABLE IF NOT EXISTS `storage`
(
    `id`         INTEGER PRIMARY KEY AUTOINCREMENT,
    `text`       TEXT,
    `file_name`  TEXT,
    `file_path`  TEXT,
    `count_text` INTEGER      NOT NULL DEFAULT 1,
    `count_file` INTEGER      NOT NULL DEFAULT 1,
    `hash_text`  VARCHAR(64)  NOT NULL,
    `hash_name`  VARCHAR(64)  NOT NULL,
    `salt_text`  VARCHAR(256) NOT NULL,
    `salt_name`  VARCHAR(256) NOT NULL,
    `salt_file`  VARCHAR(256) NOT NULL,
    `created`    DATETIME     NOT NULL,
    `updated`    DATETIME     NOT NULL,
    `expired`    DATETIME     NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS `hash_text` ON `storage` (`hash_text`);
CREATE INDEX IF NOT EXISTS `expired` ON `storage` (`expired`);

/*
id - unique identifier
text - encrypted text message
file_path - relative path to an encrypted file
file_name - encrypted file name
count_text - usage text counter (item is to be deleted if it is less than one)
count_file - usage file counter (item is to be deleted if it is less than one)
hash_text - hash of text message
hash_name - hash of text file name
salt_name - random salt for file name
salt_file - random salt for file content
created - timestamp of item create
updated - timestamp of item update
updated - timestamp of item expiration
 */