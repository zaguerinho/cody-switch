---
description: "Transcribe a meeting recording and extract structured requirements, decisions, and action items. Use this when the user provides an audio/video file from a meeting, call, or interview — or when you detect a recording that needs transcription in the current workflow."
argument-hint: "<file_path> [--language en|es] [--model turbo|medium|small] [--output <path>] [--prompt <text>] [--json <path>] [--raw] [--status]"
allowed-tools:
  - Bash(mlx_whisper *)
  - Bash(whisper *)
  - Bash(ffprobe *)
  - Bash(ffmpeg *)
  - Bash(ls *)
  - Bash(wc *)
  - Bash(du *)
  - Bash(mkdir *)
  - Bash(realpath *)
  - Bash(basename *)
  - Bash(date *)
  - Bash(stat *)
  - Bash(which *)
  - Bash(cat *)
  - Bash(tail *)
  - Bash(rm *)
  - Bash(python3 *)
  - Bash(pip *)
  - Bash(pip3 *)
  - Bash(brew *)
  - Bash(git *)
  - Bash(grep *)
  - Read
  - Glob
  - Write
  - Edit
  - AskUserQuestion
---

# Meeting Transcription & Requirements Extractor

You transcribe meeting recordings using Whisper and then analyze the transcript to extract structured requirements, decisions, and action items.

**Engine selection**: On Apple Silicon Macs, use **mlx-whisper** (native MLX framework, no MPS bugs). On all other platforms, fall back to **openai-whisper** (PyTorch, CPU). Engine detection happens automatically in Step 1a.

## Arguments

`$ARGUMENTS`

**Format**: `<file_path> [options]`

- `file_path` — Path to the audio or video file. **Required** (unless using `--status` or `--json`).
- `--language <code>` — Language code (`en`, `es`, `pt`, etc.). Default: auto-detect (Whisper chooses).
- `--model <name>` — Whisper model: `turbo` (default, fast+accurate), `medium`, `small`, `base`. Default: `turbo`.
- `--output <path>` — Directory to save output files. Default: `docs/recordings/` in the project root (auto-created, auto-gitignored). Outside a git repo, defaults to the same directory as the input file.
- `--prompt <text>` — Domain vocabulary hint passed to Whisper's `--initial_prompt`. Improves accuracy for jargon, product names, and technical terms.
- `--json <path>` — Skip transcription; analyze an existing Whisper JSON file directly (jumps to Step 3).
- `--raw` — Only transcribe, skip the analysis phase. Outputs the raw transcript only.
- `--status` — Show progress of a running transcription. No other arguments needed.

**Examples**:
```
/transcribe ~/recordings/kickoff-meeting.mp4
/transcribe ~/Desktop/call.m4a --language es
/transcribe meeting.mp3 --model medium --output ./docs/other/
/transcribe standup.wav --raw
/transcribe --status
/transcribe meeting.mp3 --prompt "Elemento, ProPMGMT, tenant onboarding, RBAC"
/transcribe --json docs/recordings/kickoff-meeting.json
```

## Step 0: Status Check (if `--status`)

If `$ARGUMENTS` contains `--status`:

1. Look for `.transcribe-progress` files in the default output directory (`docs/recordings/` if in a git repo, otherwise current directory) and any `--output` directory if provided.
2. Read the progress file. It contains: `pid`, `file`, `model`, `start_time`, `stage`, `duration_seconds`.
3. Check if the process is still running:
   ```bash
   kill -0 <pid> 2>/dev/null
   ```
4. Report status:
   - **Running**: "Transcribing `<file>` with `<model>` model — started <elapsed> ago (stage: <stage>)"
   - **Completed**: "Transcription complete. Output: `<output_files>`"
   - **Failed/stale**: "Process <pid> is no longer running. Check output directory for partial results."
   - **No progress file**: "No transcription in progress."
5. **STOP here** — do not proceed to other steps.

## Step 1: Validate Input

1. Parse `$ARGUMENTS` into file_path and options.
2. If `$ARGUMENTS` is empty (and no `--status` or `--json`), use `AskUserQuestion`:
   "Please provide the path to the audio/video file you want to transcribe. Example: `/transcribe ~/recordings/meeting.mp4`"

### 1a. Pre-flight checks

Detect the best available Whisper engine and verify dependencies.

**Step 1: Detect engine**

```bash
# Check for mlx-whisper (preferred on Apple Silicon)
which mlx_whisper 2>/dev/null
# Check for openai-whisper (fallback)
which whisper 2>/dev/null
# Check for ffprobe/ffmpeg (required by both)
which ffprobe 2>/dev/null
which ffmpeg 2>/dev/null
```

**Engine priority:**
1. **mlx-whisper** (`mlx_whisper` command) — preferred on Apple Silicon. Native MLX framework, no MPS bugs, ~50% faster than PyTorch MPS.
2. **openai-whisper** (`whisper` command) — fallback for non-Apple-Silicon or when mlx-whisper is unavailable. Runs on CPU only (MPS has multiple unfixed bugs).

Store the chosen engine as a variable (`ENGINE=mlx` or `ENGINE=openai`) for use in Step 2.

**Step 2: Install missing dependencies**

If neither whisper engine is found, or ffprobe/ffmpeg are missing, **do not stop** — collect all missing dependencies and prompt the user:

1. Build a list of missing tools and their install commands:
   - No whisper engine → recommend `pip install mlx-whisper` on Apple Silicon, `pip install openai-whisper` otherwise
   - ffprobe/ffmpeg missing → `brew install ffmpeg` (macOS) or `apt install ffmpeg` (Linux)
2. Show the user what's missing and the install commands
3. Use `AskUserQuestion`: "Missing dependencies: <list>. Want me to install them now? [Y/n]"
4. If yes, run the install commands. If no, stop with instructions for manual install.
5. After installation, re-verify with `which` to confirm.

### 1b. File validation

3. Resolve the file path to absolute using `realpath`.
4. Verify the file exists and check its size with `du -h`.
5. Probe the file for duration and format:
   ```bash
   ffprobe -v quiet -show_entries format=duration,format_name -of csv=p=0 "<file>"
   ```
6. Extract the input stem (filename without extension) for naming outputs:
   ```bash
   basename "<file>" | sed 's/\.[^.]*$//'
   ```

### 1c. Detect recording date

Determine the recording date using this priority order:

1. **Parse from filename** — look for common date patterns:
   - `YYYY-MM-DD` (e.g., `meeting-2026-03-15.mp4`)
   - `YYYYMMDD` (e.g., `20260315_standup.wav`)
   - `MM-DD-YYYY` or `MMDDYYYY`
   - `Month-DD` or `DD-Month` (e.g., `kickoff-March-15.m4a`)
2. **Extract from file metadata** via ffprobe:
   ```bash
   ffprobe -v quiet -show_entries format_tags=creation_time -of csv=p=0 "<file>"
   ```
3. **Fall back to file modification time**:
   ```bash
   stat -f "%Sm" -t "%Y-%m-%d" "<file>"   # macOS
   ```

Store the detected date as `YYYY-MM-DD` format. This is used for output filenames and the report header.

### 1d. Show file summary

Display a summary before proceeding:
```
File:     kickoff-meeting.mp4
Size:     124 MB
Duration: 47:23
Date:     2026-03-15 (from filename)
Engine:   mlx-whisper (Apple Silicon native)
Model:    turbo
Language: auto-detect
Output:   docs/recordings/
```

If the file is longer than 3 hours, warn the user and confirm they want to proceed.
If the file is longer than 15 minutes, note that transcription may take a few minutes.

### 1e. Determine output directory

7. If `--output` was provided, use that path.
8. Otherwise:
   - **Inside a git repo**: default to `docs/recordings/` relative to `git rev-parse --show-toplevel`.
   - **Outside a git repo**: default to the same directory as the input file.
9. Create the output directory if it doesn't exist: `mkdir -p "<output_dir>"`.

### 1f. Auto-gitignore (git repos only)

Only if inside a git repo:
```bash
git check-ignore -q docs/recordings/ 2>/dev/null
```
If exit code != 0 (not ignored), use the **Edit tool** to append to `.gitignore`:
```
# Meeting transcripts and recordings (large/sensitive files)
docs/recordings/
```
Inform the user that the folder was added to `.gitignore`.

### 1g. If `--json` was provided

Skip directly to **Step 3** using the provided JSON file path. Resolve the path with `realpath`, verify it exists, and extract the input stem from the JSON filename. The recording date detection still applies if a corresponding audio file can be found, otherwise use the JSON file's mtime.

## Step 2: Transcribe

### 2a. Write progress file

Before starting transcription, write a progress state file so `--status` can track it:
```bash
cat > "<output_dir>/.transcribe-progress" << EOF
pid=$$
file=<input_filename>
engine=<mlx|openai>
model=<model>
start_time=$(date +%s)
stage=transcribing
duration_seconds=<duration_from_ffprobe>
EOF
```

### 2b. Model name mapping

The user specifies short model names (`turbo`, `medium`, `small`, `base`). Map them to the correct identifier for each engine:

| User shorthand | mlx-whisper model path | openai-whisper model name |
|---|---|---|
| `turbo` (default) | `mlx-community/whisper-large-v3-turbo` | `turbo` |
| `large-v3` | `mlx-community/whisper-large-v3-mlx` | `large-v3` |
| `medium` | `mlx-community/whisper-medium-mlx-fp32` | `medium` |
| `small` | `mlx-community/whisper-small-mlx` | `small` |
| `base` | `mlx-community/whisper-base-mlx` | `base` |

### 2c. Run transcription

Build the command based on the engine detected in Step 1a. Note: **omit `--language` entirely** when language is auto-detect — neither engine accepts `--language auto`.

#### If ENGINE=mlx (mlx-whisper)

```bash
mlx_whisper "<file_path>" \
  --model <mlx_model_path> \
  --output-dir "<output_dir>" \
  --output-format json \
  --word-timestamps True \
  --hallucination-silence-threshold 2.0 \
  --verbose False
```

Add these flags conditionally:
- If `--language` was specified: add `--language <code>`
- If `--prompt` was specified: add `--initial-prompt "<text>"`

**Note:** mlx-whisper uses **hyphens** in flags (`--word-timestamps`, `--output-dir`), not underscores.

#### If ENGINE=openai (openai-whisper)

```bash
whisper "<file_path>" \
  --model <model> \
  --output_dir "<output_dir>" \
  --output_format json \
  --word_timestamps True \
  --hallucination_silence_threshold 2.0 \
  --verbose False
```

Add these flags conditionally:
- If `--language` was specified: add `--language <code>`
- If `--prompt` was specified: add `--initial_prompt "<text>"`

**IMPORTANT: Do NOT use `--device mps` with openai-whisper.** PyTorch MPS has multiple unfixed bugs with Whisper (NaN logits in fp16, float64 crashes in DTW, word timestamp failures). Always let openai-whisper run on CPU — it's slower but reliable.

**Note:** openai-whisper uses **underscores** in flags (`--word_timestamps`, `--output_dir`).

#### Bash timeout

Set timeout based on file duration: `min(duration_seconds * 10, 600000)` milliseconds. This gives ~10x real-time headroom, capped at 10 minutes. mlx-whisper on Apple Silicon completes most recordings well within this.

#### Model guidance

- **turbo** (default): Best balance of speed and accuracy. Uses `large-v3-turbo`.
- **medium**: Good fallback if turbo produces artifacts or for lower-resource machines.
- **small**: Use for quick drafts or very long recordings where speed matters.
- **base**: Only for quick-and-dirty checks.
- On **Apple Silicon** with mlx-whisper, turbo on a 60-minute recording completes in ~2-5 minutes.
- On **CPU** with openai-whisper, the same recording takes 30-60 minutes.

#### Error handling

- If transcription fails with a format error: try converting first, then retry:
  ```bash
  ffmpeg -i "<file>" -ar 16000 -ac 1 "/tmp/whisper_input_$(date +%s).wav"
  ```
  Use the converted file, then clean up the temp file after transcription completes.
- If it fails with out-of-memory: suggest a smaller model.

### 2d. Update progress file

After transcription completes, update the progress file:
```bash
sed -i '' 's/stage=transcribing/stage=analyzing/' "<output_dir>/.transcribe-progress"
```

The JSON output file is at: `<output_dir>/<input_stem>.json` (both engines produce the same JSON structure).

## Step 3: Read and Validate Transcript

1. Read the generated JSON file from `<output_dir>/<input_stem>.json`.
2. Extract segments with timestamps and text.
3. Quick quality check:
   - If many segments have very low `avg_logprob` (< -1.0) or high `no_speech_prob` (> 0.6), warn that the audio quality may have affected accuracy.
   - If the transcript is suspiciously short relative to the audio duration, flag it.
4. Build a clean, timestamped transcript:
   ```
   [00:00:15] First thing we need to discuss is the tenant onboarding flow...
   [00:01:42] Right, so the requirement is that new tenants should be able to...
   ```
5. Save the transcript as a separate file: `<output_dir>/<date>_<input_stem>_transcript.md`

If the `--raw` flag was set, output the timestamped transcript, update progress to `done`, and **STOP here**.

## Step 4: Analyze for Requirements

Now analyze the full transcript. Read it carefully and extract the following categories. Be thorough — meetings often bury requirements in casual language.

### Categories to Extract

**1. Requirements** — Things that need to be built, changed, or delivered.
- Classify each as: `MUST` (explicit requirement), `SHOULD` (strongly implied), or `COULD` (nice-to-have / suggested)
- Include the timestamp and who said it (if identifiable)
- Quote the relevant part of the transcript
- If the source segment had low confidence (`avg_logprob` < -1.0), mark with a warning note

**2. Decisions Made** — Things that were agreed upon or decided during the meeting.
- What was decided
- Any conditions or caveats mentioned

**3. Action Items** — Tasks assigned or volunteered for.
- What needs to be done
- Who is responsible (if mentioned)
- Any deadline mentioned

**4. Open Questions** — Things that were raised but NOT resolved.
- The question or ambiguity
- Any partial answers or opinions expressed

**5. Technical Constraints** — Any technical limitations, dependencies, or integration points mentioned.

**6. Assumptions** — Things that were assumed but not explicitly validated.

### Analysis Guidelines

- **Listen for requirement signals**: "we need", "it should", "the user expects", "make sure", "don't forget", "it has to", "requirement is", "they want", "by [date]"
- **Listen for decision signals**: "let's go with", "we agreed", "the plan is", "we'll do", "that's decided", "okay so we're doing"
- **Listen for hedging/ambiguity**: "maybe", "I think", "not sure", "we could", "possibly", "let me check" — these indicate open questions, NOT firm requirements
- **Separate opinions from requirements**: "I think we should use Redis" is an opinion. "The client requires sub-100ms response times" is a requirement.
- **Identify speakers when possible**: If you can infer from context (names mentioned, role references), tag items with who said them. Note: Whisper does not natively support speaker diarization — identification is best-effort from transcript context only.

## Step 5: Generate Output

### 5a. Write the structured output file

Save to `<output_dir>/<date>_<input_stem>_requirements.md`:

For example: `docs/recordings/2026-03-15_kickoff-meeting_requirements.md`

```markdown
# Meeting Transcript Analysis
**Source**: <original filename>
**Recording date**: <detected date from Step 1c>
**Analyzed**: <today's date>
**Duration**: <duration from ffprobe>
**Language**: <detected or specified language>
**Model**: <whisper model used>

---

## Summary
<2-4 sentence summary of what the meeting was about and key outcomes>

## Requirements

| # | Priority | Requirement | Source (timestamp) | Notes |
|---|----------|-------------|-------------------|-------|
| R1 | MUST | ... | [12:34] | ... |
| R2 | SHOULD | ... | [15:20] | ... |

## Decisions

| # | Decision | Context | Timestamp |
|---|----------|---------|-----------|
| D1 | ... | ... | [08:15] |

## Action Items

| # | Action | Owner | Deadline | Timestamp |
|---|--------|-------|----------|-----------|
| A1 | ... | ... | ... | [22:10] |

## Open Questions

| # | Question | Partial Context | Timestamp |
|---|----------|----------------|-----------|
| Q1 | ... | ... | [30:45] |

## Technical Constraints
- ...

## Assumptions (require validation)
- ...

---

*Full transcript: [<date>_<input_stem>_transcript.md](<date>_<input_stem>_transcript.md)*
```

### 5b. Update progress and clean up

1. Update the progress file to `done`:
   ```bash
   sed -i '' 's/stage=analyzing/stage=done/' "<output_dir>/.transcribe-progress"
   ```
2. Remove any temporary conversion files (e.g., `/tmp/whisper_input_*.wav`).
3. Remove the progress file (transcription is complete).

### 5c. Display summary to user

Show the user:
1. A brief summary of the meeting
2. The requirements table
3. Action items
4. Open questions
5. Paths to the output files:
   - Requirements: `<date>_<input_stem>_requirements.md`
   - Transcript: `<date>_<input_stem>_transcript.md`
   - Whisper JSON: `<input_stem>.json`

## Rules

- **Accuracy over completeness**: If you're unsure whether something is a requirement, classify it as an open question rather than a firm requirement.
- **Preserve original language**: When quoting the transcript, keep the original wording. Don't "clean up" what people said in the quotes — paraphrase only in the requirement description column.
- **Timestamps are anchors**: Always include timestamps so the user can go back and listen to the original context.
- **Don't invent requirements**: Only extract what was actually said. If the transcript is ambiguous, say so.
- **Flag poor audio sections**: If parts of the transcript are clearly garbled or low-confidence, flag them so the user can manually review those sections.
- **Large files**: For recordings over 1 hour, consider mentioning to the user that they can use `--model small` for a faster first pass, then re-run specific sections with `turbo` for accuracy.
- **Bilingual meetings**: If the meeting switches languages, note it. Whisper handles this reasonably well with auto-detection.
