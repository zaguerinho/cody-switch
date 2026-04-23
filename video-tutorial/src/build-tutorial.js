#!/usr/bin/env node

/**
 * build-tutorial.js — Main orchestrator.
 *
 * Usage:
 *   node build-tutorial.js --topic "..." --files "src/*.py" [--voice <id>] [--output <dir>]
 *   node build-tutorial.js --assemble-only <slug>
 *
 * Environment:
 *   ELEVENLABS_API_KEY  — required
 *   ELEVENLABS_VOICE_ID — default voice (overridable via --voice)
 *   ANTHROPIC_API_KEY   — required
 */

const fs = require('fs');
const path = require('path');
const { generateManifest, flattenSegments } = require('./generate-manifest');
const { synthesizeAll } = require('./elevenlabs');
const { assemble } = require('./assemble');

// --- Minimal glob: expand simple patterns like "src/*.py" ---
function expandGlob(pattern, cwd) {
  const { execSync } = require('child_process');
  try {
    const result = execSync(`find ${cwd} -path "${pattern}" -type f 2>/dev/null`, {
      encoding: 'utf-8',
      cwd
    });
    return result.trim().split('\n').filter(Boolean);
  } catch {
    return [];
  }
}

function slugify(text) {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
    .substring(0, 50);
}

function parseArgs(argv) {
  const args = {
    topic: null,
    files: null,
    voice: process.env.ELEVENLABS_VOICE_ID || null,
    output: null,
    assembleOnly: null
  };

  for (let i = 2; i < argv.length; i++) {
    switch (argv[i]) {
      case '--topic':
        args.topic = argv[++i];
        break;
      case '--files':
        args.files = argv[++i];
        break;
      case '--voice':
        args.voice = argv[++i];
        break;
      case '--output':
        args.output = argv[++i];
        break;
      case '--assemble-only':
        args.assembleOnly = argv[++i];
        break;
      default:
        if (!args.topic && !argv[i].startsWith('--')) {
          args.topic = argv[i];
        }
    }
  }

  return args;
}

function requireEnv(name) {
  const val = process.env[name];
  if (!val) {
    process.stderr.write(`Error: ${name} environment variable is required\n`);
    process.exit(1);
  }
  return val;
}

async function main() {
  const args = parseArgs(process.argv);
  const rootDir = path.resolve(__dirname, '..');
  const templateDir = path.join(rootDir, 'template');
  const browserDir = path.join(rootDir, 'browser');
  const outputBase = args.output || path.join(rootDir, 'output', 'tutorials');

  // --- Assemble-only mode ---
  if (args.assembleOnly) {
    const slug = args.assembleOnly;
    const tutorialDir = path.join(outputBase, slug);
    const manifestPath = path.join(tutorialDir, 'manifest.json');
    const synthPath = path.join(tutorialDir, 'synth-results.json');

    if (!fs.existsSync(manifestPath)) {
      process.stderr.write(`Error: manifest not found at ${manifestPath}\n`);
      process.exit(1);
    }
    if (!fs.existsSync(synthPath)) {
      process.stderr.write(`Error: synth results not found at ${synthPath}\n`);
      process.exit(1);
    }

    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf-8'));
    const synthResults = JSON.parse(fs.readFileSync(synthPath, 'utf-8'));
    const sourceDir = manifest._sourceDir || process.cwd();

    const outputPath = path.join(tutorialDir, 'tutorial.html');
    assemble({ templateDir, browserDir, manifest, synthResults, sourceDir, outputPath });
    process.stderr.write(`\nDone! Open: ${outputPath}\n`);
    notify('Video Tutorial Ready', `"${manifest.title}" has been re-assembled.`, outputPath);
    return;
  }

  // --- Full build mode ---
  if (!args.topic) {
    process.stderr.write('Usage: node build-tutorial.js --topic "..." --files "pattern"\n');
    process.stderr.write('       node build-tutorial.js --assemble-only <slug>\n');
    process.exit(1);
  }

  const anthropicKey = requireEnv('ANTHROPIC_API_KEY');
  const elevenLabsKey = requireEnv('ELEVENLABS_API_KEY');
  if (!args.voice) {
    process.stderr.write('Error: ELEVENLABS_VOICE_ID env var or --voice flag required\n');
    process.exit(1);
  }

  // Resolve source files
  const cwd = process.cwd();
  const filePattern = args.files || './**/*.{py,js,ts,php,go,rb}';
  const filePaths = expandGlob(filePattern, cwd);

  if (filePaths.length === 0) {
    process.stderr.write(`Error: no files matched pattern "${filePattern}" in ${cwd}\n`);
    process.exit(1);
  }

  process.stderr.write(`\n=== Video Tutorial Builder ===\n`);
  process.stderr.write(`Topic: ${args.topic}\n`);
  process.stderr.write(`Files: ${filePaths.length} files matched\n\n`);

  // Setup output directory
  const slug = slugify(args.topic);
  const tutorialDir = path.join(outputBase, slug);
  fs.mkdirSync(tutorialDir, { recursive: true });

  // Step 1: Generate manifest
  process.stderr.write(`--- Step 1: Generate manifest ---\n`);
  const manifest = await generateManifest(args.topic, filePaths, anthropicKey);
  manifest._sourceDir = cwd; // Store for assemble-only mode
  fs.writeFileSync(
    path.join(tutorialDir, 'manifest.json'),
    JSON.stringify(manifest, null, 2),
    'utf-8'
  );
  process.stderr.write(`Manifest saved to ${tutorialDir}/manifest.json\n\n`);

  // Step 2: Synthesize audio (with incremental save + resume)
  process.stderr.write(`--- Step 2: Synthesize audio ---\n`);
  const segments = flattenSegments(manifest);
  const synthPath = path.join(tutorialDir, 'synth-results.json');

  // Load existing partial results if available
  let synthResults = [];
  const completedIds = new Set();
  if (fs.existsSync(synthPath)) {
    try {
      const existing = JSON.parse(fs.readFileSync(synthPath, 'utf-8'));
      if (Array.isArray(existing) && existing.length > 0) {
        synthResults = existing;
        existing.forEach(r => completedIds.add(r.id));
        process.stderr.write(`  Resuming: ${completedIds.size}/${segments.length} segments already synthesized\n`);
      }
    } catch (e) {
      process.stderr.write(`  Warning: could not parse existing synth results, starting fresh\n`);
    }
  }

  const remainingSegments = segments
    .map(s => ({ id: s.id, text: s.text }))
    .filter(s => !completedIds.has(s.id));

  if (remainingSegments.length > 0) {
    const newResults = await synthesizeAll(
      remainingSegments,
      args.voice,
      elevenLabsKey,
      {},
      (done, total, id) => {
        process.stderr.write(`  [${completedIds.size + done}/${segments.length}] ${id}\n`);
      },
      (result) => {
        // Incremental save after each segment so partial progress survives crashes
        synthResults.push(result);
        try {
          fs.writeFileSync(synthPath, JSON.stringify(synthResults, null, 2), 'utf-8');
        } catch (e) {
          process.stderr.write(`  Warning: incremental save failed: ${e.message}\n`);
        }
      }
    );

    // If onResult already populated synthResults, avoid duplicates
    // (newResults is the same data, synthResults already has it via onResult)
  }

  // Final save (ensures complete file even if incremental saves had issues)
  fs.writeFileSync(synthPath, JSON.stringify(synthResults, null, 2), 'utf-8');
  process.stderr.write(`Synth results saved (${synthResults.length} segments)\n\n`);

  // Step 3: Assemble
  process.stderr.write(`--- Step 3: Assemble ---\n`);
  const outputPath = path.join(tutorialDir, 'tutorial.html');
  assemble({ templateDir, browserDir, manifest, synthResults, sourceDir: cwd, outputPath });

  process.stderr.write(`\n=== Done! ===\n`);
  process.stderr.write(`Tutorial: ${outputPath}\n`);
  process.stderr.write(`Manifest: ${tutorialDir}/manifest.json\n`);
  process.stderr.write(`\nOpen in browser: open "${outputPath}"\n`);

  // macOS desktop notification
  notify('Video Tutorial Ready', `"${manifest.title}" has been assembled.`, outputPath);
}

/**
 * Send a macOS desktop notification. Falls back silently on other platforms.
 */
function notify(title, message, openPath) {
  const { execSync } = require('child_process');
  if (process.platform !== 'darwin') return;

  try {
    // Try terminal-notifier first (richer notifications with click-to-open)
    execSync(`which terminal-notifier`, { stdio: 'ignore' });
    execSync(`terminal-notifier -title ${JSON.stringify(title)} -message ${JSON.stringify(message)} -open ${JSON.stringify('file://' + openPath)} -sound default -group video-tutorial`, { stdio: 'ignore' });
  } catch {
    // Fall back to osascript (always available on macOS)
    try {
      execSync(`osascript -e 'display notification ${JSON.stringify(message)} with title ${JSON.stringify(title)} sound name "Glass"'`, { stdio: 'ignore' });
    } catch {
      // Notification failed — not critical
    }
  }
}

main().catch(err => {
  process.stderr.write(`\nFatal error: ${err.message}\n`);
  if (err.stack) process.stderr.write(`${err.stack}\n`);
  notify('Video Tutorial Failed', err.message, '');
  process.exit(1);
});
