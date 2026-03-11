# Contributing

Thanks for your interest in adb-tui.

## Reporting Issues

- Search existing issues before opening a new one.
- For bugs, include: OS, Go version, ADB version, steps to reproduce, and the error output.
- For feature requests, describe the use case.

## Pull Requests

1. Open an issue first for non-trivial changes.
2. Fork the repo and create a branch from `main`.
3. Run `make check` before submitting — it covers formatting, vet, lint, and tests.
4. Keep PRs focused. One change per PR.

## Development

```bash
make build        # Build
make test-race    # Test with race detector
make lint         # Lint
make check        # All quality checks
```

## Code Style

- Follow standard Go conventions (`gofmt -s`).
- Use `ShellArgs` (parameterized) over `Shell` (string concat) for all new ADB commands.
- Keep comments minimal — code should be self-explanatory.
