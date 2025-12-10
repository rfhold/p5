# Changelog

## v0.6.0

### Features
* 93a3c132619bf235c5a29f42dad43444269f9054 feat: add Grafana builtin plugin with resource URL opener
* f84b5bc191b186347ca842b561fc2eeb8ac80077 feat: add vim-style / filter to all list components
* 51312509f4522d7571f985802a384138fbc2858d feat: pass exclude URNs to Pulumi SDK operations
### Refactoring
* 362a32ae52db5ba0d306b2783b870a5fa32dc707 refactor: use Pulumi SDK for imports and cleanup dead code

## v0.5.1

### Bug Fixes
* 2c1b64561cf97a33e163b915e80feea834208fec fix: stack selection pre auth and auth locking

## v0.5.0

### Features
* cd8b8d3eb607a847bfb5b6d945a96c9f7a43ab34 feat: expose secrets provider to auth plugins

## v0.4.0

### Features
* 7497c9943decd41d390ec84aad04101cd2c3d325 feat: add OpenTelemetry integration with slog logging
### Bug Fixes
* 522de0dbd7b9e00b31839cc68ea1a9eabb43b056 fix: add Logger to test Dependencies to prevent nil pointer panic
* 2b8db6dc2679e1f1783d8909cb1e67c8f0163822 fix: resolve flaky integration tests with done indicator
* a63354b76add58023dd679eb13b057bee242d2d2 fix: resource sequence ordering
### Refactoring
* 7da83fa5dd0ea3b973d5b3c71057bb25b26359e0 refactor: linter issues

## v0.3.1

### Bug Fixes
* 94b38c232b33f2fec946b0f5b727c122ebea4751 fix: env variable handling to avoid setting the process env

## v0.3.0

### Features
* 690e306b20cdbaf7be96fc35c5ed5f112dc4e170 feat: resource opener & k9s builtin
### Other
* 569f01f7f5d58348dda0f95570f1698b98151bc4 build: add protoc plugins as go tool dependencies
* 04c8e320da5b55e2854721da6ab52c520b25e433 chore: auto-commit changelog updates in release script
* 313bf41d85a64832f4a25d8b0b1db530ec292272 chore: comment cleanup and agent/contributing doc update
* 9f6228175ae03862c803feaca4f4baba6d080149 chore: go mod tidy

## v0.2.0

### Features
* 150bd9769bdad673fca494d09454d2757c470ba2 feat: provider inputs to import plugins
### Bug Fixes
* c126a2cbc8c5cd1123fd42ad8519a860c97416ba fix: context in async operations
* bf79892cd6622c23ff0d74a369662053a52f5319 fix: improve release script changelog generation
* 00dbddaff479494306373bc5126cabbacc0d6ee6 fix: modal dismiss focus change
### Other
* 6d59234a22ce775b5c3380a2592904c8b46a5022 chore: comment cleanup
* a7f0ce30ae151b0bc20cc0368e3439b07d34ec6f chore: releaser and changelogs
* 62bda74b8329d033c7743bd92e4d50e537f7505b chore: remove old dev-env example plugin
