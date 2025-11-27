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
- [ ] Cross-compilation works (`make ci-cross-compile`)
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

## Pipeline Stages

The unified CI/CD pipeline runs in three stages:

1. **VALIDATE** (runs on all PRs): Lint → Test → Security → Build Verification
2. **BUILD** (main branch only): Cross-platform release builds
3. **RELEASE** (main branch + `[release]` flag): GitHub Release creation

**All tests must pass** - No test failures are ignored.

See [CICD.md](CICD.md) for detailed pipeline architecture.