#!/usr/bin/env python3
"""Reproduce two support mistakes using disposable local fixtures (Python 3.10+)."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import time
import zipfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for tool in ("diagsift", "restic", "rclone"):
        parser.add_argument("--" + tool, required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path,
                        help="New directory; existing paths are refused")
    args = parser.parse_args()
    binaries = {name: getattr(args, name).resolve(strict=True)
                for name in ("diagsift", "restic", "rclone")}
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=False)
    root = Path(__file__).resolve().parents[1]
    env = {"PATH": os.pathsep.join(dict.fromkeys(str(p.parent) for p in binaries.values()))
           + os.pathsep + os.defpath, "HOME": str(output), "LANG": "C.UTF-8"}
    # Windows needs these to start ordinary native executables.
    for key in ("SystemRoot", "WINDIR", "TEMP", "TMP"):
        if key in os.environ:
            env[key] = os.environ[key]
    events = []

    def clean(text):
        text = text.replace(str(output), "<WORK>")
        for name, path in binaries.items():
            text = text.replace(str(path), name)
        return text

    def run(cmd, cwd, expected=0, stdin=None):
        start = time.monotonic()
        result = subprocess.run([str(x) for x in cmd], cwd=cwd, env=env,
                                input=stdin, text=True, stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT, timeout=90)
        event = {"command": clean(subprocess.list2cmdline([str(x) for x in cmd])),
                 "cwd": clean(str(cwd)), "input": stdin,
                 "exit_code": result.returncode,
                 "duration_seconds": round(time.monotonic() - start, 4),
                 "output": clean(result.stdout)}
        events.append(event)
        if expected is None:
            assert result.returncode != 0, event
        else:
            assert result.returncode == expected, event
        return result.stdout

    versions = {name: run([path, "version"], output).strip()
                for name, path in binaries.items()}
    summaries = []
    for name in ("restic", "rclone"):
        work = output / name
        work.mkdir()
        selected = work / "selected"
        selected.mkdir()
        shutil.copyfile(root / "examples" / name / "diagsift.yaml",
                        selected / "diagsift.yaml")
        if name == "restic":
            pwfile = work / "fixture-passphrase.txt"
            pwfile.write_text("Disposable local demonstration phrase only.\n")
            pwfile.chmod(0o600)
            repo = work / "repository"
            run([binaries[name], "-r", repo, "-p", pwfile, "init"], work)
            failure = run([binaries[name], "-r", repo, "-p",
                           "not-a-password-file", "snapshots"], work, expected=None)
            marker = "not-a-password-file"
            correction = run([binaries[name], "-r", repo, "-p", pwfile,
                              "snapshots"], work)
            context = "Local fixture. Failing command: restic -r repository -p not-a-password-file snapshots\n"
            source = "https://forum.restic.net/t/fatal-wrong-password-or-no-key-found/3125/11"
        else:
            data = work / "data"
            data.mkdir()
            (data / "hello.txt").write_text("Local demonstration file.\n")
            config = work / "fixture.conf"
            config.write_text("[demo]\ntype = alias\nremote = " + data.as_posix() + "\n")
            failure = run([binaries[name], "--config", config, "ls", "demo", "-vv"],
                          work, expected=None)
            marker = "directory not found"
            correction = run([binaries[name], "--config", config, "ls", "demo:"], work)
            assert "hello.txt" in correction
            context = "Local fixture. Failing command: rclone --config fixture.conf ls demo -vv\n"
            source = "https://forum.rclone.org/t/configuration-settings-return-error-error-listing-directory-not-found/19874"
        assert marker in failure, failure
        (selected / "diagnostic.log").write_text(context + clean(failure))
        manifest = selected / "diagsift.yaml"
        run([binaries["diagsift"], "validate", manifest], selected)
        run([binaries["diagsift"], "plan", manifest], selected)
        bundle = selected / "case.zip"
        # This is an automated fixture exercise. The transcript records its YES input.
        run([binaries["diagsift"], "collect", manifest, "--output", bundle], selected,
            stdin="YES\n")
        inspected = json.loads(run([binaries["diagsift"], "inspect", bundle], selected))
        assert inspected["hashesVerified"] is True and inspected["partial"] is False
        assert inspected["errorCount"] == 0, inspected
        with zipfile.ZipFile(bundle) as archive:
            names = archive.namelist()
            assert "errors.json" not in names
            collector_names = [n for n in names if n.startswith("collectors/")]
            assert len(collector_names) == 3, names
            logs = [n for n in collector_names if n.startswith("collectors/log/")]
            assert len(logs) == 1
            collected = archive.read(logs[0]).decode()
            assert marker in collected, collected
            assert not any("fixture.conf" in n or "passphrase" in n for n in names)
        (work / "collected-log.txt").write_text(collected)
        summaries.append({"case": name, "source": source,
                          "classification": "maintainer-run local reproduction; not independent adoption",
                          "collector_entries": collector_names, "inspect": inspected,
                          "diagnostic_marker_preserved": marker,
                          "correction_exit_code": 0,
                          "collected_log": collected})
    result = {"recorded_date": time.strftime("%Y-%m-%d", time.gmtime()),
              "platform": platform.platform(), "versions": versions,
              "binary_sha256": {n: hashlib.sha256(p.read_bytes()).hexdigest()
                                for n, p in binaries.items()},
              "scope": "local fixtures only; no remote service; no user data; no time-saving claim",
              "cases": summaries, "events": events}
    (output / "results.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps({"result": "PASS", "cases": len(summaries),
                      "results": str(output / "results.json")}, indent=2))


if __name__ == "__main__":
    main()
