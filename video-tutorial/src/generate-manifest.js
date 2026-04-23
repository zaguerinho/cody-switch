/**
 * Content generator — build-time module.
 *
 * Calls Claude API to produce a scene manifest JSON from a topic description
 * and relevant source files. Co-generates narration and visual cues in a
 * single call to prevent drift.
 *
 * Output: { title, chapters: [{ id, title, segments: [{ id, narration, cue }] }] }
 */

const https = require('https');
const fs = require('fs');
const path = require('path');

const ANTHROPIC_BASE = 'api.anthropic.com';
const ANTHROPIC_VERSION = '2023-06-01';
const MODEL = 'claude-opus-4-5-20250514';
const MAX_TOKENS = 8192;

/**
 * Make an HTTPS POST request to the Anthropic API.
 */
function anthropicPost(apiKey, body) {
  return new Promise((resolve, reject) => {
    const payload = JSON.stringify(body);
    const options = {
      hostname: ANTHROPIC_BASE,
      port: 443,
      path: '/v1/messages',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-api-key': apiKey,
        'anthropic-version': ANTHROPIC_VERSION,
        'Content-Length': Buffer.byteLength(payload)
      }
    };

    const req = https.request(options, (res) => {
      const chunks = [];
      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => {
        const raw = Buffer.concat(chunks).toString();
        if (res.statusCode >= 400) {
          reject(new Error(`Anthropic API error ${res.statusCode}: ${raw.substring(0, 500)}`));
          return;
        }
        try {
          resolve(JSON.parse(raw));
        } catch (e) {
          reject(new Error(`Failed to parse Anthropic response: ${raw.substring(0, 200)}`));
        }
      });
    });

    req.on('error', reject);
    req.write(payload);
    req.end();
  });
}

/**
 * Read source files and format them for the prompt.
 *
 * @param {Array<string>} filePaths - Absolute paths to source files
 * @returns {string} Formatted source code block
 */
function formatSourceFiles(filePaths) {
  const parts = [];
  for (const fp of filePaths) {
    if (!fs.existsSync(fp)) {
      throw new Error(`Source file not found: ${fp}`);
    }
    const content = fs.readFileSync(fp, 'utf-8');
    const lines = content.split('\n');
    const numbered = lines.map((line, i) => `${(i + 1).toString().padStart(4)} | ${line}`).join('\n');
    parts.push(`=== ${path.basename(fp)} (${fp}) ===\n${numbered}\n`);
  }
  return parts.join('\n');
}

/**
 * Build the system prompt for manifest generation.
 */
function buildSystemPrompt() {
  return `You are a technical tutorial scriptwriter. You produce structured JSON scene manifests for narrated code tutorials.

RULES:
- Output ONLY valid JSON. No markdown fences, no explanation, no preamble.
- Every narration sentence MUST reference only code visible in its paired cue.
- Never reference code in narration that isn't in the current cue's line range.
- Each chapter should contain 60-90 seconds of narration (roughly 150-225 words).
- Each segment should be 1-3 sentences of narration.
- Use conversational but precise technical language.
- Code references must use exact file paths and valid line ranges from the provided sources.
- Line ranges are inclusive: [start, end] means lines start through end.

OUTPUT SCHEMA:
{
  "title": "string — tutorial title",
  "chapters": [
    {
      "id": "ch1",
      "title": "string — chapter title",
      "segments": [
        {
          "id": "ch1-s1",
          "narration": "string — what the narrator says",
          "cue": {
            "type": "highlight",
            "file": "string — relative file path exactly as provided",
            "lines": [startLine, endLine],
            "scroll_to": startLine
          }
        }
      ]
    }
  ]
}`;
}

/**
 * Build the user prompt from topic and source files.
 */
function buildUserPrompt(topic, sourceCode) {
  return `Create a narrated code tutorial about:

${topic}

SOURCE FILES:
${sourceCode}

Generate a scene manifest JSON with 3-6 chapters that walk through this code, explaining what it does and how it works. Each chapter should focus on a specific aspect or section of the code.

Remember:
- Every segment's narration must describe ONLY the code in its cue's line range
- Use the exact file paths shown above
- Line numbers must be within the file's actual range
- Co-generate narration and cues together — they must be in sync`;
}

/**
 * Validate the generated manifest against the actual source files.
 *
 * @param {Object} manifest - The generated manifest
 * @param {Array<string>} filePaths - Source file paths
 * @throws {Error} If validation fails
 */
function validateManifest(manifest, filePaths) {
  const errors = [];

  if (!manifest.title || typeof manifest.title !== 'string') {
    errors.push('Missing or invalid "title" field');
  }
  if (!Array.isArray(manifest.chapters) || manifest.chapters.length === 0) {
    errors.push('Missing or empty "chapters" array');
  }

  // Build a map of file paths → line counts
  const fileLinesMap = {};
  for (const fp of filePaths) {
    const content = fs.readFileSync(fp, 'utf-8');
    const lineCount = content.split('\n').length;
    fileLinesMap[path.basename(fp)] = lineCount;
    fileLinesMap[fp] = lineCount;
  }

  for (const chapter of (manifest.chapters || [])) {
    if (!chapter.id || !chapter.title) {
      errors.push(`Chapter missing id or title: ${JSON.stringify(chapter).substring(0, 100)}`);
    }
    for (const seg of (chapter.segments || [])) {
      if (!seg.id || !seg.narration) {
        errors.push(`Segment missing id or narration in chapter ${chapter.id}`);
        continue;
      }
      const cue = seg.cue;
      if (!cue || !cue.file || !cue.lines) {
        errors.push(`Segment ${seg.id}: missing cue, file, or lines`);
        continue;
      }

      // Check file exists
      const lineCount = fileLinesMap[cue.file] || fileLinesMap[path.basename(cue.file)];
      if (!lineCount) {
        errors.push(`Segment ${seg.id}: file "${cue.file}" not found in source files`);
        continue;
      }

      // Check line range
      const [start, end] = cue.lines;
      if (start < 1 || end < start) {
        errors.push(`Segment ${seg.id}: invalid line range [${start}, ${end}]`);
      }
      if (end > lineCount) {
        errors.push(`Segment ${seg.id}: line ${end} exceeds file length (${lineCount} lines) in "${cue.file}"`);
      }
    }
  }

  if (errors.length > 0) {
    throw new Error(`Manifest validation failed:\n${errors.map(e => `  - ${e}`).join('\n')}`);
  }
}

/**
 * Generate a scene manifest from a topic and source files.
 *
 * @param {string} topic - Topic description
 * @param {Array<string>} filePaths - Absolute paths to relevant source files
 * @param {string} apiKey - Anthropic API key
 * @param {Object} [options]
 * @param {string} [options.model] - Model to use (default: claude-opus-4-5)
 * @param {boolean} [options.skipValidation] - Skip file/line validation
 * @returns {Promise<Object>} The scene manifest
 */
async function generateManifest(topic, filePaths, apiKey, options = {}) {
  const { model = MODEL, skipValidation = false } = options;

  process.stderr.write(`Generating manifest for: "${topic.substring(0, 80)}..."\n`);
  process.stderr.write(`Source files: ${filePaths.map(f => path.basename(f)).join(', ')}\n`);

  const sourceCode = formatSourceFiles(filePaths);
  const systemPrompt = buildSystemPrompt();
  const userPrompt = buildUserPrompt(topic, sourceCode);

  const response = await anthropicPost(apiKey, {
    model,
    max_tokens: MAX_TOKENS,
    system: systemPrompt,
    messages: [{ role: 'user', content: userPrompt }]
  });

  // Extract text content from response
  const textBlock = response.content.find(b => b.type === 'text');
  if (!textBlock) {
    throw new Error('No text content in Anthropic response');
  }

  // Parse JSON — strip any accidental markdown fences
  let jsonText = textBlock.text.trim();
  if (jsonText.startsWith('```')) {
    jsonText = jsonText.replace(/^```(?:json)?\n?/, '').replace(/\n?```$/, '');
  }

  let manifest;
  try {
    manifest = JSON.parse(jsonText);
  } catch (e) {
    throw new Error(`Failed to parse manifest JSON:\n${jsonText.substring(0, 500)}\n\nParse error: ${e.message}`);
  }

  // Validate
  if (!skipValidation) {
    validateManifest(manifest, filePaths);
    process.stderr.write(`Manifest validated: ${manifest.chapters.length} chapters, ${manifest.chapters.reduce((n, ch) => n + ch.segments.length, 0)} segments\n`);
  }

  return manifest;
}

/**
 * Flatten manifest segments into an ordered array for synthesis.
 *
 * @param {Object} manifest - The scene manifest
 * @returns {Array<{ id: string, text: string, chapterId: string, cue: Object }>}
 */
function flattenSegments(manifest) {
  const segments = [];
  for (const chapter of manifest.chapters) {
    for (const seg of chapter.segments) {
      segments.push({
        id: seg.id,
        text: seg.narration,
        chapterId: chapter.id,
        cue: seg.cue
      });
    }
  }
  return segments;
}

module.exports = {
  generateManifest,
  validateManifest,
  flattenSegments,
  formatSourceFiles
};
