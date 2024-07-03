# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.2] - 2024-07-03
### Fixed
- Comments with mentions in direct messages was not delivered to client.

## [1.2.1] - 2023-12-03

### Changed

- Migrate to telegram-bot-api/v5

## [1.2.0] - 2023-12-02

### Added

- Like/Unlike buttons to comment notifications to like/unlike comments.

### Changed

- "Subscribe to comments"/"Unsubscribe from comments" buttons now uses the new
  FreeFeed API for comments notifications.

### Removed

- It is no longer possible to subscribe to comments by sending the bot a link to
  the post.
