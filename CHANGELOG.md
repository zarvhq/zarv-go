# Zarv Go - Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `pkg/ingestion`: client for the ingestion gateway. `New` validates
  `ZARV_INGESTION_URL` and `ZARV_INGESTION_KEY`; `Send` posts the gateway's
  body with the Bearer key, retries 503/5xx/429 and transport errors honoring
  `Retry-After`, and returns the gateway's refusals, a 202's included, as
  `RefusedError`
- Initial repository structure
- Middleware package with authentication and authorization
- Comprehensive documentation
- MIT License

### Changed
- Module name updated to follow Go conventions (github.com/zarvhq/zarv-go)

## [1.0.0] - 2026-02-07

### Added
- Authenticate middleware for Fiber applications
- AuthProfile struct and GetAuthProfile function
- Authorization helper methods (IsZarvAdmin, IsUserAdmin, IsViewer)
- Support for internal service requests
- Header-based authentication validation
