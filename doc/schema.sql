CREATE TABLE IF NOT EXISTS `storage`
(
    `id`         INTEGER PRIMARY KEY AUTOINCREMENT,
    `key`        VARCHAR(64),
    `text`       TEXT,
    `file_meta`  TEXT,
    `file_path`  TEXT,
    `count_text` INTEGER      NOT NULL DEFAULT 1,
    `count_file` INTEGER      NOT NULL DEFAULT 1,
    `hash_text`  VARCHAR(64)  NOT NULL,
    `hash_meta`  VARCHAR(64)  NOT NULL,
    `hash_file`  VARCHAR(64)  NOT NULL,
    `salt_text`  VARCHAR(256) NOT NULL,
    `salt_meta`  VARCHAR(256) NOT NULL,
    `salt_file`  VARCHAR(256) NOT NULL,
    `created`    DATETIME     NOT NULL,
    `updated`    DATETIME     NOT NULL,
    `expired`    DATETIME     NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS `key` ON `storage` (`key`);
CREATE INDEX IF NOT EXISTS `expired` ON `storage` (`expired`,`count_text`,`count_file`);

/*
id - unique identifier
key - random unique identifier (url part - request UUID)
text - encrypted text message
file_path - relative path to an encrypted file
file_meta - encrypted file meta data, JSON {name, size, content_type}
count_text - usage text counter (item is to be deleted if it is less than one)
count_file - usage file counter (item is to be deleted if it is less than one)
hash_text - hash of text message
hash_meta - hash of text file name
hash_file - hash of file
salt_meta - random salt for file name
salt_file - random salt for file content
created - timestamp of item create
updated - timestamp of item update
updated - timestamp of item expiration
 */