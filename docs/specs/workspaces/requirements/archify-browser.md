---
status: draft
system: workspaces
created: 2026-09-17
owners:
  - kandev
---

# Archify Browser Requirements

## Overview

Kandev shall provide a stable browser entry for a repository's Git-native Archify diagrams. The
workspace system owns this capability because repository selection, Git revision resolution,
source cleanliness, and authorization determine which architecture sources are visible. Desktop
and phone presentation are part of the same repository-bound outcome.

## Terminology

- **Architecture binding:** A repository, Git ref, repository-relative source directory, and
  trusted Archify runtime selected for architecture browsing.
- **Resolved revision:** The immutable commit SHA produced from the configured Git ref.
- **Derived render:** HTML or validation evidence generated from source at the resolved revision
  and retained only as replaceable cache data.

## Requirements

### REQ-WORKSPACES-ARCHIFY-BROWSER-001: Browse repository-owned Archify diagrams

**Intent:** Give developers a stable Kandev destination for inspecting architecture sources and
their validated renderings without duplicating the architecture model.

**User story:** As a developer, I want to browse a repository's Archify diagrams from Kandev, so
that architecture evidence remains accessible independently of task sessions.

#### Acceptance criteria

- **AC-WORKSPACES-ARCHIFY-BROWSER-001.1:** When an authorized workspace contains one configured
  architecture binding, opening `/architecture` shall resolve its repository and configured Git
  ref to one commit and display the repository, configured ref, resolved SHA, relative source path,
  and source cleanliness.
- **AC-WORKSPACES-ARCHIFY-BROWSER-001.2:** When the configured source directory contains Archify
  source files, the page shall list each supported source with its diagram title, type, source path,
  and validation status, and shall allow switching between diagrams without a task session.
- **AC-WORKSPACES-ARCHIFY-BROWSER-001.3:** Selecting a diagram shall expose both its exact source
  JSON at the resolved revision and a browser render produced from those source bytes. Existing
  generated files in the repository shall not become authoritative inputs.
- **AC-WORKSPACES-ARCHIFY-BROWSER-001.4:** When a validated render is absent or refresh is
  requested, Kandev shall validate and regenerate it reproducibly from the resolved revision. It
  shall treat all generated HTML and receipts as disposable cache and shall not write them to the
  repository or commit them.
- **AC-WORKSPACES-ARCHIFY-BROWSER-001.5:** The page shall expose loading, missing configuration,
  repository/ref/path/runtime failures, structured validation failures, and render failures with a
  retry path. It shall not silently select another repository, revision, path, or runtime.
- **AC-WORKSPACES-ARCHIFY-BROWSER-001.6:** Refresh shall resolve the configured ref again, replace
  the visible inventory atomically, and keep the previously selected diagram only when the same
  source remains present. One response shall never mix source or render data from different SHAs.
- **AC-WORKSPACES-ARCHIFY-BROWSER-001.7:** On phone, `/architecture` shall provide the same source,
  validation, render, Git metadata, refresh, and navigation outcomes through a focused diagram
  surface. Required actions shall have a minimum 44-pixel active touch dimension and the document
  shall not gain horizontal overflow.
- **AC-WORKSPACES-ARCHIFY-BROWSER-001.8:** Only users authorized for the owning workspace shall read
  the inventory, source, validation result, or render, or request regeneration. Source paths and
  runtime execution shall remain confined to the configured repository and runtime.

## Out of scope

- Editing or saving Archify source.
- Semantic versus visual change classification.
- Automatic commits, task lifecycle hooks, before/after diffs, or review automation.
- OmniRoute configuration or changes to TheBrain product behavior or production data.
- A second architecture database, copied Archify JSON, or a separately operated Archify server.
