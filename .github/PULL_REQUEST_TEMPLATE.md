## Description

Brief description of the changes in this PR.

## Type of Change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Refactoring (no functional changes)
- [ ] Performance improvement
- [ ] Test improvement

## Testing

Run full CI suite locally before creating PR:
```bash
make ci-full
```

Or run individual checks:
- [ ] Linting passes (`make ci-lint`)
- [ ] All tests pass (`make ci-test`)
- [ ] Security checks pass (`make ci-security`)
- [ ] Release build works (`make release-local`)
- [ ] Manual testing completed (describe below)

**Manual Testing:**
<!-- Describe any manual testing you performed -->

## Checklist

- [ ] Code follows the project's style guidelines
- [ ] Self-review of the code completed
- [ ] Code is commented where necessary
- [ ] Documentation updated (if applicable)
- [ ] No breaking changes (or breaking changes are documented)
- [ ] Tests added/updated for new functionality

## Screenshots (if applicable)

<!-- Add screenshots for UI changes -->

## Additional Context

<!-- Add any other context about the pull request here -->

---

## CI

`ci.yml` runs these in parallel, gated on a path filter, with a final `ci-ok`
job that always reports:

```
changes ──┬──> lint ───────────────┐
          ├──> test (matrix) ──────┤
          ├──> security ───────────┼──> ci-ok
          └──> test-install-script ┘
```

**All tests must pass** - No test failures are ignored.

Releases are triggered by pushing a `v*` tag, not by merging to main.
`release.yml` calls `ci.yml` first, so a tag cannot publish untested code.

See [Development Guide](../docs/development.md#cicd-pipeline) for details.