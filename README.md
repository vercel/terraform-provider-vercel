# terraform-provider-vercel

Terraform provider for Vercel

## Required Software

- Go v1.17 or later
- Taskfile https://taskfile.dev/

## Development

Development overriding of the provider can be set up via

```bash
task install
```

This sets up a ~/.terraformrc that will point the vercel terraform provider to your local codebase.

To continually rebuild the binary on any changes, run:

```bash
task build --watch
```
