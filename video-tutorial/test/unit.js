#!/usr/bin/env node

/**
 * Unit tests for video-tutorial pure functions.
 * No external dependencies — uses Node's built-in assert.
 *
 * Run: node test/unit.js
 */

const assert = require('assert');
const path = require('path');
const fs = require('fs');

const { characterTimestampsToWords } = require('../src/elevenlabs');
const { tokenizeLine, buildSyncData, buildCueMap } = require('../src/assemble');
const { validateManifest, flattenSegments } = require('../src/generate-manifest');

let passed = 0;
let failed = 0;

function test(name, fn) {
  try {
    fn();
    passed++;
    process.stdout.write(`  \x1b[32m✓\x1b[0m ${name}\n`);
  } catch (err) {
    failed++;
    process.stdout.write(`  \x1b[31m✗\x1b[0m ${name}\n`);
    process.stdout.write(`    ${err.message}\n`);
  }
}

// =============================================================================
// characterTimestampsToWords
// =============================================================================
process.stdout.write('\ncharacterTimestampsToWords\n');

test('groups characters into words', () => {
  const result = characterTimestampsToWords({
    characters: ['H','e','l','l','o',' ','w','o','r','l','d'],
    character_start_times_seconds: [0,0.1,0.15,0.2,0.25,0.3,0.35,0.4,0.45,0.5,0.55],
    character_end_times_seconds:   [0.1,0.15,0.2,0.25,0.3,0.35,0.4,0.45,0.5,0.55,0.6]
  });
  assert.strictEqual(result.length, 2);
  assert.strictEqual(result[0].word, 'Hello');
  assert.strictEqual(result[0].start, 0);
  assert.strictEqual(result[0].end, 0.3);
  assert.strictEqual(result[1].word, 'world');
  assert.strictEqual(result[1].start, 0.35);
  assert.strictEqual(result[1].end, 0.6);
});

test('handles single word', () => {
  const result = characterTimestampsToWords({
    characters: ['O','K'],
    character_start_times_seconds: [0, 0.1],
    character_end_times_seconds: [0.1, 0.2]
  });
  assert.strictEqual(result.length, 1);
  assert.strictEqual(result[0].word, 'OK');
});

test('handles trailing space', () => {
  const result = characterTimestampsToWords({
    characters: ['H','i',' '],
    character_start_times_seconds: [0, 0.1, 0.2],
    character_end_times_seconds: [0.1, 0.2, 0.3]
  });
  assert.strictEqual(result.length, 1);
  assert.strictEqual(result[0].word, 'Hi');
});

test('handles multiple spaces between words', () => {
  const result = characterTimestampsToWords({
    characters: ['a',' ',' ','b'],
    character_start_times_seconds: [0, 0.1, 0.2, 0.3],
    character_end_times_seconds: [0.1, 0.2, 0.3, 0.4]
  });
  assert.strictEqual(result.length, 2);
  assert.strictEqual(result[0].word, 'a');
  assert.strictEqual(result[1].word, 'b');
});

test('handles empty input', () => {
  const result = characterTimestampsToWords({
    characters: [],
    character_start_times_seconds: [],
    character_end_times_seconds: []
  });
  assert.strictEqual(result.length, 0);
});

test('handles punctuation attached to words', () => {
  const result = characterTimestampsToWords({
    characters: ['H','i','.',' ','O','K','!'],
    character_start_times_seconds: [0,0.1,0.2,0.3,0.4,0.5,0.6],
    character_end_times_seconds:   [0.1,0.2,0.3,0.4,0.5,0.6,0.7]
  });
  assert.strictEqual(result.length, 2);
  assert.strictEqual(result[0].word, 'Hi.');
  assert.strictEqual(result[1].word, 'OK!');
});

// =============================================================================
// tokenizeLine
// =============================================================================
process.stdout.write('\ntokenizeLine\n');

test('tokenizes keywords', () => {
  const html = tokenizeLine('const x = 5;');
  assert(html.includes('tk-keyword'), 'should contain keyword class');
  assert(html.includes('const'), 'should contain "const"');
});

test('tokenizes strings', () => {
  const html = tokenizeLine('const s = "hello";');
  assert(html.includes('tk-string'), 'should contain string class');
  assert(html.includes('hello'), 'should contain string content');
});

test('tokenizes single-quoted strings', () => {
  const html = tokenizeLine("const s = 'world';");
  assert(html.includes('tk-string'));
  assert(html.includes('world'));
});

test('tokenizes numbers', () => {
  const html = tokenizeLine('const n = 42;');
  assert(html.includes('tk-number'));
  assert(html.includes('42'));
});

test('tokenizes hex numbers', () => {
  const html = tokenizeLine('const mask = 0xFF;');
  assert(html.includes('tk-number'));
  assert(html.includes('0xFF'));
});

test('tokenizes function calls', () => {
  const html = tokenizeLine('console.log("hi");');
  assert(html.includes('tk-function'));
  assert(html.includes('log'));
});

test('tokenizes line comments //', () => {
  const html = tokenizeLine('// comment here');
  assert(html.includes('tk-comment'));
  assert(html.includes('comment here'));
});

test('tokenizes Python # comments', () => {
  const html = tokenizeLine('  # Python comment');
  assert(html.includes('tk-comment'));
});

test('tokenizes block comment opening', () => {
  const html = tokenizeLine('/* block comment */');
  assert(html.includes('tk-comment'));
});

test('tokenizes template literals with interpolation', () => {
  const html = tokenizeLine('const msg = `hello ${name}`;');
  assert(html.includes('tk-string'));
  assert(html.includes('hello'));
  assert(html.includes('name'));
});

test('tokenizes triple-quoted strings', () => {
  const html = tokenizeLine('doc = """docstring"""');
  assert(html.includes('tk-string'));
  assert(html.includes('docstring'));
});

test('tokenizes decorators', () => {
  const html = tokenizeLine('@app.route("/api")');
  assert(html.includes('tk-function'));
  assert(html.includes('@app.route'));
});

test('escapes HTML entities', () => {
  const html = tokenizeLine('if (a < b && c > d) {}');
  assert(html.includes('&lt;'), 'should escape <');
  assert(html.includes('&gt;'), 'should escape >');
  assert(html.includes('&amp;'), 'should escape &');
});

test('handles empty line', () => {
  const html = tokenizeLine('');
  assert.strictEqual(html, '');
});

test('handles whitespace-only line', () => {
  const html = tokenizeLine('    ');
  assert.strictEqual(html, '    ');
});

test('inline comment after code', () => {
  const html = tokenizeLine('const x = 1; // set x');
  assert(html.includes('tk-keyword'), 'should have keyword');
  assert(html.includes('tk-number'), 'should have number');
  assert(html.includes('tk-comment'), 'should have comment');
});

// =============================================================================
// buildSyncData
// =============================================================================
process.stdout.write('\nbuildSyncData\n');

test('builds flat word array with correct offsets', () => {
  const manifest = {
    chapters: [
      { id: 'ch1', title: 'One', segments: [
        { id: 's1', narration: 'Hello', cue: { type: 'highlight', file: 'a.py', lines: [1, 5], scroll_to: 1 } },
      ]},
      { id: 'ch2', title: 'Two', segments: [
        { id: 's2', narration: 'World', cue: { type: 'highlight', file: 'a.py', lines: [6, 10], scroll_to: 6 } },
      ]},
    ]
  };
  const synthResults = [
    { id: 's1', wordTimestamps: [{ word: 'Hello', start: 0, end: 0.5 }], duration: 0.6 },
    { id: 's2', wordTimestamps: [{ word: 'World', start: 0, end: 0.5 }], duration: 0.6 },
  ];

  const result = buildSyncData(manifest, synthResults);

  assert.strictEqual(result.words.length, 2);
  assert.strictEqual(result.words[0].word, 'Hello');
  assert.strictEqual(result.words[0].start, 0);
  assert.strictEqual(result.words[1].word, 'World');
  assert(result.words[1].start > 0.5, 'second word should be offset');
  assert.strictEqual(result.chapters.length, 2);
  assert.strictEqual(result.chapters[0].startWord, 0);
  assert.strictEqual(result.chapters[1].startWord, 1);
  assert.strictEqual(result.wordToCue[0], 's1');
  assert.strictEqual(result.wordToCue[1], 's2');
  assert(result.totalDuration > 1, 'total should account for gaps');
});

test('handles empty manifest', () => {
  const result = buildSyncData({ chapters: [] }, []);
  assert.strictEqual(result.words.length, 0);
  assert.strictEqual(result.chapters.length, 0);
  assert.strictEqual(result.totalDuration, 0);
});

// =============================================================================
// buildCueMap
// =============================================================================
process.stdout.write('\nbuildCueMap\n');

test('maps segment IDs to cues', () => {
  const manifest = {
    chapters: [{
      id: 'ch1', title: 'Ch1',
      segments: [
        { id: 's1', narration: 'A', cue: { type: 'highlight', file: 'a.py', lines: [1, 5], scroll_to: 1 } },
        { id: 's2', narration: 'B', cue: { type: 'reveal', file: 'b.py', lines: [10, 20], scroll_to: 10 } },
      ]
    }]
  };
  const map = buildCueMap(manifest);
  assert.strictEqual(Object.keys(map).length, 2);
  assert.strictEqual(map['s1'].file, 'a.py');
  assert.strictEqual(map['s2'].type, 'reveal');
  assert.deepStrictEqual(map['s2'].lines, [10, 20]);
});

// =============================================================================
// validateManifest
// =============================================================================
process.stdout.write('\nvalidateManifest\n');

test('passes valid manifest', () => {
  // Create temp source file
  const tmpDir = fs.mkdtempSync(path.join(require('os').tmpdir(), 'vt-test-'));
  const srcFile = path.join(tmpDir, 'test.py');
  fs.writeFileSync(srcFile, 'line1\nline2\nline3\nline4\nline5\n');

  const manifest = {
    title: 'Test Tutorial',
    chapters: [{
      id: 'ch1', title: 'Chapter 1',
      segments: [{
        id: 'ch1-s1', narration: 'Hello',
        cue: { type: 'highlight', file: srcFile, lines: [1, 5], scroll_to: 1 }
      }]
    }]
  };

  assert.doesNotThrow(() => validateManifest(manifest, [srcFile]));
  fs.rmSync(tmpDir, { recursive: true });
});

test('fails on missing title', () => {
  assert.throws(
    () => validateManifest({ chapters: [{ id: 'ch1', title: 'X', segments: [] }] }, []),
    /title/i
  );
});

test('fails on empty chapters', () => {
  assert.throws(
    () => validateManifest({ title: 'T', chapters: [] }, []),
    /chapters/i
  );
});

test('fails on out-of-bounds line range', () => {
  const tmpDir = fs.mkdtempSync(path.join(require('os').tmpdir(), 'vt-test-'));
  const srcFile = path.join(tmpDir, 'small.py');
  fs.writeFileSync(srcFile, 'line1\nline2\n');

  const manifest = {
    title: 'Test',
    chapters: [{
      id: 'ch1', title: 'Ch1',
      segments: [{
        id: 's1', narration: 'Hi',
        cue: { type: 'highlight', file: srcFile, lines: [1, 99], scroll_to: 1 }
      }]
    }]
  };

  assert.throws(
    () => validateManifest(manifest, [srcFile]),
    /exceeds file length/i
  );
  fs.rmSync(tmpDir, { recursive: true });
});

test('fails on nonexistent file reference', () => {
  // Create a real file so validateManifest can read it, but reference a different file in the cue
  const tmpDir = fs.mkdtempSync(path.join(require('os').tmpdir(), 'vt-test-'));
  const realFile = path.join(tmpDir, 'real.py');
  fs.writeFileSync(realFile, 'line1\nline2\nline3\n');

  const manifest = {
    title: 'Test',
    chapters: [{
      id: 'ch1', title: 'Ch1',
      segments: [{
        id: 's1', narration: 'Hi',
        cue: { type: 'highlight', file: 'nonexistent.py', lines: [1, 5], scroll_to: 1 }
      }]
    }]
  };

  assert.throws(
    () => validateManifest(manifest, [realFile]),
    /not found/i
  );
  fs.rmSync(tmpDir, { recursive: true });
});

// =============================================================================
// flattenSegments
// =============================================================================
process.stdout.write('\nflattenSegments\n');

test('flattens chapters into ordered segment array', () => {
  const manifest = {
    chapters: [
      { id: 'ch1', title: 'A', segments: [
        { id: 's1', narration: 'First', cue: { file: 'a.py', lines: [1, 5] } },
        { id: 's2', narration: 'Second', cue: { file: 'a.py', lines: [6, 10] } },
      ]},
      { id: 'ch2', title: 'B', segments: [
        { id: 's3', narration: 'Third', cue: { file: 'b.py', lines: [1, 3] } },
      ]},
    ]
  };

  const flat = flattenSegments(manifest);
  assert.strictEqual(flat.length, 3);
  assert.strictEqual(flat[0].id, 's1');
  assert.strictEqual(flat[0].text, 'First');
  assert.strictEqual(flat[0].chapterId, 'ch1');
  assert.strictEqual(flat[2].id, 's3');
  assert.strictEqual(flat[2].chapterId, 'ch2');
});

test('handles empty manifest', () => {
  const flat = flattenSegments({ chapters: [] });
  assert.strictEqual(flat.length, 0);
});

// =============================================================================
// Summary
// =============================================================================
process.stdout.write(`\n${passed + failed} tests, \x1b[32m${passed} passed\x1b[0m`);
if (failed > 0) {
  process.stdout.write(`, \x1b[31m${failed} failed\x1b[0m`);
}
process.stdout.write('\n\n');
process.exit(failed > 0 ? 1 : 0);
