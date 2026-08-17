# Roadmap

## v0.1 local release candidate

- Complete M0-M5 security, portability, and documentation gates.
- Obtain explicit human maintainer approval before publishing anything.

## Pilot gate

The generic-manifest adoption hypothesis remains unproven. Before broader polish,
seek one independent maintainer who reviews and ships a real manifest, a second
independent maintainer/project integration, and one non-maintainer who generates
and inspects a bundle. Record genuine feedback; do not manufacture issues, users,
downloads, stars, or pull requests.

## Possible later work (not v0.1 commitments)

- Evaluate Windows Job Objects for stronger descendant cleanup.
- Evaluate race-resistant root-relative file opening where supported.
- Evaluate bounded directory and glob collection only if pilot evidence supports it.
- Revise the alpha manifest only from real pilot evidence.

Upload, telemetry, accounts, AI diagnosis, plugins, elevation, arbitrary shell,
and broad filesystem discovery remain out of scope.
