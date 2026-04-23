---
name: video-tutorial
description: >
  Generate narrated eLearning screencast tutorials as self-contained HTML files.
  Use when the user says "/video-tutorial", "create tutorial", "screencast",
  or wants to generate a code walkthrough with narrated audio.
  Requires Codex CLI. TTS via Piper (free/local) or ElevenLabs (cloud).
---

# /video-tutorial

Generate a narrated eLearning screencast tutorial as a self-contained HTML file.

## Usage

```
/video-tutorial <topic> [--files <glob>] [--tts piper|elevenlabs|auto]
/video-tutorial <topic> --tts piper [--piper-model en_US-lessac-medium]
/video-tutorial <topic> --tts elevenlabs [--voice <elevenlabs_voice_id>]
/video-tutorial --assemble-only <slug>
```

## Process

### Step 1: Parse arguments

Extract from the user's invocation:
- `topic` — the tutorial subject (required unless `--assemble-only`)
- `--files <glob>` — file pattern to include (default: auto-detect from project)
- `--voice <id>` — ElevenLabs voice ID (default: `$ELEVENLABS_VOICE_ID`)
- `--assemble-only <slug>` — re-assemble from existing manifest without re-generating

If no topic is provided and not `--assemble-only`, ask:
> What topic should the tutorial cover? Include which files or modules to focus on.

### Step 2: Validate environment

Check that required tools are available:

```bash
# Check Claude CLI (used for manifest generation)
which claude

# Check TTS backend
which piper       # Free/local option
echo "ELEVENLABS_API_KEY: ${ELEVENLABS_API_KEY:+set}"  # Cloud option
```

The binary auto-detects the best available TTS:
- If `ELEVENLABS_API_KEY` is set → uses ElevenLabs (higher quality, word-level sync)
- If `piper` is installed → uses Piper (free, local, estimated word sync)
- If neither → shows install instructions for both options

If Piper is not installed and the user wants the free option:
```
pip install piper-tts
```

If the user prefers ElevenLabs:
```
export ELEVENLABS_API_KEY=your-key
export ELEVENLABS_VOICE_ID=your-voice-id
```

**No ANTHROPIC_API_KEY needed** — manifest generation uses the Claude CLI directly.

Stop here if validation fails. Do not proceed without at least one TTS backend.

### Step 3: Identify source files

If `--files` was specified, use that glob pattern. Otherwise, auto-detect:

1. Read the project structure to identify relevant files for the topic
2. Prefer files the user mentioned in the topic description
3. Limit to 5-8 files maximum (more files = longer audio = larger HTML)
4. Exclude test files, config files, and generated code unless specifically relevant

Present the file list to the user for confirmation:
> I'll include these files in the tutorial:
> - `src/auth/login.py` (45 lines)
> - `src/auth/middleware.py` (62 lines)
>
> Look good, or should I add/remove any?

### Step 4: Locate the video-tutorial binary

Check for the installed binary first, then fall back to the repo copy:

```bash
# Check installed location (from cody-switch install --extras)
TUTORIAL_BIN="$HOME/.claude/bin/video-tutorial"
if [ ! -x "$TUTORIAL_BIN" ]; then
  # Fall back to repo location
  SCRIPT_DIR="$(dirname "$(readlink -f "$(command -v cody-switch)")")"
  TUTORIAL_BIN="$SCRIPT_DIR/video-tutorial/video-tutorial"
fi
ls "$TUTORIAL_BIN"
```

If not found in either location, tell the user:
> The video-tutorial binary isn't installed. Run:
> `cody-switch install --skill video-tutorial`

### Step 5: Detect TTS backend

Check what's available:

```bash
echo "ELEVENLABS_API_KEY: ${ELEVENLABS_API_KEY:+set}"
which piper 2>/dev/null || python3 -m piper --help >/dev/null 2>&1 && echo "piper: installed"
```

If ElevenLabs key is set, use ElevenLabs (higher quality). If only Piper is available, use Piper (free). If neither, tell the user their options and stop.

### Step 6: Voice selection (ElevenLabs only)

**IMPORTANT: The binary's interactive prompts don't work inside Codex. YOU must handle them.**

If using ElevenLabs, present the voice options to the user:

> **Choose a voice for the tutorial narration:**
>
> | # | Voice | Gender | Accent | Style |
> |---|-------|--------|--------|-------|
> | 1 | **Adam** | Male | American | Deep, authoritative — great for technical narration |
> | 2 | **Antoni** | Male | American | Warm, conversational — friendly and approachable |
> | 3 | **Arnold** | Male | American | Strong, confident — clear and commanding |
> | 4 | **Rachel** | Female | American | Calm, clear — professional and polished |
> | 5 | **Domi** | Female | American | Energetic, expressive — engaging delivery |
> | 6 | **Bella** | Female | American | Soft, gentle — soothing and articulate |
> | 7 | **Elli** | Female | American | Young, bright — upbeat and clear |
> | 8 | **Gigi** | Female | American | Animated, youthful — lively and fun |
> | 9 | **Daniel** | Male | British | Refined, articulate — BBC-style delivery |
> | 10 | **Callum** | Male | British | Smooth, measured — calm and professional |

Voice IDs (map the user's choice to the ID):
- Adam: `pNInz6obpgDQGcFmaJgB`
- Antoni: `ErXwobaYiN019PkySvjV`
- Arnold: `VR6AewLTigWG4xSOukaG`
- Rachel: `21m00Tcm4TlvDq8ikWAM`
- Domi: `AZnzlk1XvdvUeBnXmlld`
- Bella: `EXAVITQu4vr4xnSDxMaL`
- Elli: `MF3mGyEYCl7XYWbV9V6O`
- Gigi: `jBpfuIE2acCO8z3wKNLl`
- Daniel: `onwK4e9ZLuTAKqWW03F9`
- Callum: `N2lVS1w4EtoT3dr4eOWO`

If the user has `$ELEVENLABS_VOICE_ID` set, show that as the default. If they don't pick, use Adam.

### Step 7: Cost estimate and confirmation (ElevenLabs only)

Before running the build, estimate the cost:

1. Count the total characters in the source files being included
2. Multiply by ~3 (narration is roughly 3x the character count of the source, accounting for the AI-generated script)
3. Show the estimate:

> **ElevenLabs Cost Estimate:**
> - Source files: ~X characters
> - Estimated narration: ~Y characters (credits)
> - Free tier (10k/month): $0.00
> - Starter plan ($5/30k): ~$Z.ZZ
>
> Proceed?

Wait for user confirmation before running the build. If they say no, stop.

### Step 8: Run the build

Always pass `-y` to skip the binary's own interactive prompts (Claude handles them above).

**ElevenLabs:**
```bash
cd <project-root>
"$TUTORIAL_BIN" build \
  --topic "<topic>" \
  --files "<glob>" \
  --tts elevenlabs \
  --voice "<selected_voice_id>" \
  -y \
  -o "$SCRIPT_DIR/video-tutorial/output/tutorials"
```

**Piper:**
```bash
cd <project-root>
"$TUTORIAL_BIN" build \
  --topic "<topic>" \
  --files "<glob>" \
  --tts piper \
  -y \
  -o "$SCRIPT_DIR/video-tutorial/output/tutorials"
```

**Assemble-only:**
```bash
"$TUTORIAL_BIN" assemble "<slug>" -o "$SCRIPT_DIR/video-tutorial/output/tutorials"
```

Stream the stderr output to the user so they can see progress:
- Manifest generation (via Claude CLI)
- Per-segment audio synthesis
- HTML assembly

### Step 6: Handle errors

**Manifest validation failure** — the generated manifest references files or lines that don't exist:
- Show the specific validation errors
- Suggest adjusting the file list or topic description
- Offer to re-run with `--files` narrowed down

**ElevenLabs rate limit** — the binary retries automatically with exponential backoff:
- If it still fails after retries, show the error
- Suggest waiting a minute and re-running with `--assemble-only` (manifest is already saved)

**Large file warning** — if the output HTML exceeds 50MB:
- Warn the user that the file may be slow to open
- Suggest splitting into fewer chapters or excluding large source files

### Step 7: Present results

On success, show:

```
Tutorial generated!

  Open:     output/tutorials/<slug>/tutorial.html
  Manifest: output/tutorials/<slug>/manifest.json

To open in your browser:
  open output/tutorials/<slug>/tutorial.html

To re-assemble after editing the manifest:
  /video-tutorial --assemble-only <slug>
```

## Additional tools

The binary provides extra subcommands:

```bash
# Validate a manifest without building
"$TUTORIAL_BIN" validate output/tutorials/<slug>/manifest.json

# Inspect build artifacts
"$TUTORIAL_BIN" inspect <slug>

# Export transcript as SRT subtitles
"$TUTORIAL_BIN" export-transcript <slug> --format srt

# Export as WebVTT captions
"$TUTORIAL_BIN" export-transcript <slug> --format vtt
```

## Output structure

```
output/tutorials/<slug>/
  tutorial.html       ← self-contained, open in any browser
  manifest.json       ← editable scene manifest
  synth-results.json  ← cached audio + timestamps (for re-assembly)
```

## How the tutorial player works

The generated `tutorial.html` is a single file with everything inlined:

- **Code viewer** — syntax-highlighted source code with VS Code Dark+ theme
- **Audio playback** — narrated audio (base64-encoded, plays via Web Audio API)
- **Sync engine** — word-level timestamps drive code highlighting in real-time
- **Chapter navigation** — sidebar with chapter list, progress tracking, localStorage resume
- **Chatbot drawer** — pause and ask questions about the code (requires user's own API key in browser)
- **Keyboard shortcuts** — Space (play/pause), arrows (skip 5s), speed cycling

No server required. Works offline. Shareable as a single file.

## Important constraints

- **No video file** — the player renders code as live HTML, not a recorded video
- **No CDN calls** — everything is inlined (CSS, JS, audio, code)
- **Fail loudly** — if a cue references a missing file or invalid line range, the build crashes with a clear error. Never produce a broken tutorial silently.
- **Chatbot is optional** — if the viewer doesn't set an API key in their browser, the chatbot prompts for one instead of crashing
- **No ANTHROPIC_API_KEY needed** — manifest generation uses Claude CLI, not the API directly
