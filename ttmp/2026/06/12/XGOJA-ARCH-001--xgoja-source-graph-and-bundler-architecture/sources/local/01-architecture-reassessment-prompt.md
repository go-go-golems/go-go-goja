---
Title: Architecture reassessment prompt
Ticket: XGOJA-ARCH-001
Status: active
Topics:
  - goja
  - xgoja
  - typescript
  - tooling
  - developer-experience
DocType: reference
Intent: source
Summary: Captures the user request that triggered the xgoja architecture reassessment.
---

# Architecture reassessment prompt

The architecture ticket was triggered by this user prompt:

> Ok, write that architecture docuemnt, in great depth and detail.
>
> Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

The preceding discussion framed xgoja as increasingly similar to a bundler/compiler, except that it bundles Go packages that provide Go-implemented JavaScript libraries in addition to JavaScript/TypeScript source, help, assets, jsverb command metadata, and hot reload behavior. The design should therefore step back from individual TypeScript fixes and propose a source graph, import resolver, build plan, runtime plan, and provider input model for xgoja.
