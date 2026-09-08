#!/usr/bin/env python3
"""Render an 80-second replay of captured CLI output (Pillow and ffmpeg required)."""
import argparse
import json
from pathlib import Path
import subprocess
import tempfile
import textwrap
from PIL import Image, ImageDraw, ImageFont

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--record", required=True, type=Path)
parser.add_argument("--output", required=True, type=Path)
parser.add_argument("--font", required=True, type=Path)
args = parser.parse_args()
record = json.loads(args.record.read_text())
events = record["events"]
args.output.mkdir(parents=True, exist_ok=True)
for name in ("demo.mp4", "demo.gif", "demo.md"):
    if (args.output / name).exists():
        raise SystemExit("Refusing to replace " + name)


def event(index, tail=None):
    e = events[index]
    lines = e["output"].strip().splitlines()
    if tail is not None:
        lines = ["[earlier output omitted]"] + lines[-tail:]
    consent = "\n[automated stdin: YES]" if e["input"] else ""
    return "$ " + e["command"] + consent + "\n" + "\n".join(lines) + "\n[exit " + str(e["exit_code"]) + "]"


plan = json.loads(events[13]["output"])
plan_excerpt = {key: plan[key] for key in ("name", "limits")}
scenes = [
    ("01 / Verify the release", "Real CLI output. Local fixtures. Pauses edited for readability.",
     event(0) + "\n\nRecorded binary SHA-256:\n" + record["binary_sha256"]["diagsift"]
     + "\n\nMatched the published rc.2 SHA256SUMS.\nDownload: GitHub > tjacky87-lab/diagsift > Releases"),
    ("02 / Reproduce a restic mistake", "The -p argument needs a file path. The fixture file fixes this case.",
     event(4) + "\n\n" + event(5)),
    ("03 / Reproduce a rclone mistake", "The remote name needs a colon. rclone itself already explains this.",
     event(10, 4) + "\n\n" + event(11)),
    ("04 / Validate and preview", "The existing manifest collects version, OS/architecture and one log.",
     event(12) + "\n\n$ " + events[13]["command"] + "\n[plan excerpt]\n"
     + json.dumps(plan_excerpt, indent=2)
     + "\n[collector summary: rclone version; os + arch; diagnostic.log]"),
    ("05 / Explicit consent, local ZIP", "The fixture runner supplies YES. This is not an independent user test.",
     event(14, 2)),
    ("06 / Inspect the archive", "Checks structure and hashes. Does not certify safe-to-share content.",
     event(15)),
    ("07 / Read the actual collected log", "Selected ZIP entry excerpt; disposable paths normalized before collection.",
     "collectors/log/diagnostic.log\n\n" + "\n".join(
         line for line in record["cases"][1]["collected_log"].splitlines()
         if "Failing command" in line or "NOTICE:" in line or "ERROR" in line)),
    ("08 / What this evidence means", "Full commands, outputs, source links and limitations are in the repository.",
     "Two controlled cases passed.\n\nEach bundle: 3 collectors, 0 errors, partial=false.\nUseful error text survived collection.\nCorrected upstream commands succeeded.\n\nNot demonstrated: independent adoption, time saved,\ncomplete secret removal, or automatic diagnosis.\n\nRead: docs/evidence/README.md\nTry: GitHub issue #5 | Feedback: issue #4"),
]
font = ImageFont.truetype(str(args.font), 20)
title_font = ImageFont.truetype(str(args.font), 29)
small = ImageFont.truetype(str(args.font), 17)
images = []
with tempfile.TemporaryDirectory(prefix="diagsift-demo-") as temp:
    temp = Path(temp)
    for i, (title, caption, body) in enumerate(scenes):
        im = Image.new("RGB", (1280, 720), "#0b1420")
        draw = ImageDraw.Draw(im)
        draw.text((44, 26), "DIAGSIFT  /  v0.1.0-rc.2", font=small, fill="#62d9ba")
        draw.text((44, 65), title, font=title_font, fill="#f1f5f9")
        draw.rounded_rectangle((32, 125, 1248, 626), radius=15, fill="#121f30")
        lines = []
        for line in body.splitlines():
            lines.extend(textwrap.wrap(line, width=96, replace_whitespace=False,
                                       drop_whitespace=False) or [""])
        if len(lines) > 19:
            raise SystemExit(f"Scene {i + 1} needs {len(lines)} lines; limit is 19")
        for j, line in enumerate(lines):
            color = "#62d9ba" if line.startswith("$") else "#e0e7ef"
            draw.text((52, 146 + j * 24), line, font=font, fill=color)
        draw.text((44, 647), caption, font=small, fill="#aabbd0")
        draw.text((44, 681), "RECORDED OUTPUT REPLAY | " + record["recorded_date"],
                  font=small, fill="#91a2b8")
        draw.text((1090, 681), f"{i * 10:02d}-{(i + 1) * 10:02d}s", font=small, fill="#91a2b8")
        draw.rectangle((0, 712, int(1280 * (i + 1) / 8), 720), fill="#62d9ba")
        im.save(temp / f"scene-{i}.png")
        images.append(im)
    # A small looping preview; MP4 is the full-size downloadable replay.
    thumbs = [im.resize((960, 540)).quantize(colors=96) for im in images]
    thumbs[0].save(args.output / "demo.gif", save_all=True, append_images=thumbs[1:],
                   duration=10000, loop=0, optimize=True)
    listing = "".join(f"file 'scene-{i}.png'\nduration 10\n" for i in range(8))
    listing += "file 'scene-7.png'\n"
    (temp / "frames.txt").write_text(listing)
    subprocess.run(["ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "concat",
                    "-safe", "0", "-i", str(temp / "frames.txt"), "-t", "80",
                    "-vf", "fps=2", "-c:v", "libx264", "-preset", "slow", "-crf", "22",
                    "-pix_fmt", "yuv420p", "-movflags", "+faststart", "-n",
                    str(args.output / "demo.mp4")], check=True)
    images[5].save(args.output / "demo-preview.png")
transcript = "# 80-second demo transcript\n\nActual captured CLI output, replayed with edited pauses and excerpts.\n"
transcript += "This is a maintainer-run fixture exercise, not an independent user recording.\n"
for title, caption, body in scenes:
    transcript += f"\n## {title}\n\n{caption}\n\n```text\n{body}\n```\n"
transcript += "\n## Rebuild the replay\n\nRequires Python, Pillow, ffmpeg and a monospace TrueType font.\n\n```sh\npython3 scripts/render-evidence-demo.py --record docs/evidence/results.json \\\n  --output /absolute/path/to/new-demo-output \\\n  --font /absolute/path/to/DejaVuSansMono.ttf\n```\n"
(args.output / "demo.md").write_text(transcript)
print("Created 80-second MP4, GIF, preview and transcript in", args.output)
