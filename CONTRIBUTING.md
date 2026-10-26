# Contributing to OpenLinkHub

OpenLinkHub focuses on **Corsair hardware support for Linux**. Features for controlling other manufacturers' devices are outside the project's scope.

## Pull requests
- Search existing issues and pull requests to avoid duplicating work.
- For substantial features, architectural changes, or new dependencies, open an issue and discuss the approach before implementing it.
- **Keep changes small and focused.** Fix the problem without unrelated refactoring.
- Prefer existing code and architecture over new abstractions, dependencies, or persistent state.
- Explain **what** the change fixes and **why** it is needed.
- Test your changes, preferably on real hardware when device behavior is involved. Include what you tested.
- Avoid excessive comments. Document non-obvious decisions, not every line of code.
- Do not bundle unrelated refactoring, formatting, or cleanups with a bug fix.
- Reuse existing functionality instead of duplicating it.

## AI-assisted contributions
AI-assisted code is welcome, but **you are responsible for understanding, reviewing, and testing every change you submit**. Do not submit large AI-generated patches without checking whether a simpler solution exists.

If you use AI to help create a change:
- Read, understand, and critically review the entire diff before submitting it.
- Verify that the solution matches the actual problem and existing codebase.
- Remove unnecessary abstractions, excessive comments, unrelated modifications, and generated boilerplate.
- Test the behavior yourself; do not claim tests were performed if they were not.
- Be prepared to explain the purpose of each change and why a simpler approach would not work.

**Unnecessarily complex, out-of-scope, or unreviewed PRs may be closed without a line-by-line review**, regardless of how the code was written.
